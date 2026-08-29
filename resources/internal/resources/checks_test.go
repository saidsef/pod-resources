package containers

import (
	"errors"
	"sync/atomic"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

var (
	errListFailed      = errors.New("listing pods failed")
	podMetricsResource = v1beta1.SchemeGroupVersion.WithResource("pods")
)

func TestCheckResourceRL(t *testing.T) {
	tests := []struct {
		name string
		info PodInfo
		want []string
	}{
		{
			name: "nothing set at all warns on every limit and request",
			info: PodInfo{
				Name: "web-0", Container: "web", Namespace: "shop",
				Usage: UsageInfo{Name: "web", CPU: 500, Memory: 200},
			},
			want: []string{
				"WARNING: Container web in pod web-0 namespace shop has no cpu limit set. Current usage: 500m",
				"WARNING: Container web in pod web-0 namespace shop has no cpu request set. Current usage: 500m",
				"WARNING: Container web in pod web-0 namespace shop has no memory limit set. Current usage: 200Mi",
				"WARNING: Container web in pod web-0 namespace shop has no memory request set. Current usage: 200Mi",
			},
		},
		{
			name: "usage above both limit and request alerts on each",
			info: PodInfo{
				Name: "web-0", Container: "web", Namespace: "shop",
				Resources: v1.ResourceRequirements{
					Limits:   v1.ResourceList{v1.ResourceCPU: resource.MustParse("10m"), v1.ResourceMemory: resource.MustParse("15Mi")},
					Requests: v1.ResourceList{v1.ResourceCPU: resource.MustParse("5m"), v1.ResourceMemory: resource.MustParse("5Mi")},
				},
				Usage: UsageInfo{Name: "web", CPU: 9000, Memory: 4096},
			},
			want: []string{
				"ALERT: Container web in pod web-0 namespace shop has cpu usage of 9000m, above its limit of 10m",
				"ALERT: Container web in pod web-0 namespace shop has cpu usage of 9000m, above its request of 5m",
				"ALERT: Container web in pod web-0 namespace shop has memory usage of 4096Mi, above its limit of 15Mi",
				"ALERT: Container web in pod web-0 namespace shop has memory usage of 4096Mi, above its request of 5Mi",
			},
		},
		{
			name: "a cpu limit with no cpu request warns about the request only",
			info: PodInfo{
				Name: "web-0", Container: "web", Namespace: "shop",
				Resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m"), v1.ResourceMemory: resource.MustParse("64Mi")},
				},
				Usage: UsageInfo{Name: "web", CPU: 1, Memory: 1},
			},
			want: []string{
				"WARNING: Container web in pod web-0 namespace shop has no cpu request set. Current usage: 1m",
				"WARNING: Container web in pod web-0 namespace shop has no memory request set. Current usage: 1Mi",
			},
		},
		{
			name: "usage inside both limit and request says nothing",
			info: PodInfo{
				Name: "web-0", Container: "web", Namespace: "shop",
				Resources: v1.ResourceRequirements{
					Limits:   v1.ResourceList{v1.ResourceCPU: resource.MustParse("1"), v1.ResourceMemory: resource.MustParse("1Gi")},
					Requests: v1.ResourceList{v1.ResourceCPU: resource.MustParse("500m"), v1.ResourceMemory: resource.MustParse("512Mi")},
				},
				Usage: UsageInfo{Name: "web", CPU: 100, Memory: 100},
			},
			want: nil,
		},
		{
			name: "usage equal to the limit is not over it",
			info: PodInfo{
				Name: "web-0", Container: "web", Namespace: "shop",
				Resources: v1.ResourceRequirements{
					Limits:   v1.ResourceList{v1.ResourceCPU: resource.MustParse("500m"), v1.ResourceMemory: resource.MustParse("100Mi")},
					Requests: v1.ResourceList{v1.ResourceCPU: resource.MustParse("500m"), v1.ResourceMemory: resource.MustParse("100Mi")},
				},
				Usage: UsageInfo{Name: "web", CPU: 500, Memory: 100},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			CheckResourceRL(tt.info, func(message string) { got = append(got, message) })

			if len(got) != len(tt.want) {
				t.Fatalf("got %d messages, want %d\ngot:  %q\nwant: %q", len(got), len(tt.want), got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("message %d:\ngot:  %q\nwant: %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractUsageInfo(t *testing.T) {
	metrics := &v1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "web-0", Namespace: "shop"},
		Containers: []v1beta1.ContainerMetrics{
			{
				Name: "web",
				Usage: v1.ResourceList{
					v1.ResourceCPU:              resource.MustParse("250m"),
					v1.ResourceMemory:           resource.MustParse("128Mi"),
					v1.ResourceEphemeralStorage: resource.MustParse("1024"),
				},
			},
			{
				Name: "sidecar",
				Usage: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("1"),
					v1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		},
	}

	want := []UsageInfo{
		{Name: "web", CPU: 250, Memory: 128, EphemeralStorage: 1024},
		{Name: "sidecar", CPU: 1000, Memory: 1024, EphemeralStorage: 0},
	}

	got := ExtractUsageInfo(metrics)
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGetPodInfo(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-0", Namespace: "shop"},
		Spec: v1.PodSpec{Containers: []v1.Container{
			{
				Name: "web",
				Resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{v1.ResourceCPU: resource.MustParse("100m")},
				},
			},
			{
				Name: "sidecar",
				Resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{v1.ResourceMemory: resource.MustParse("64Mi")},
				},
			},
		}},
	}
	systemPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "kube-proxy-abcde", Namespace: "kube-system"},
		Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "kube-proxy"}}},
	}
	metrics := &v1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "web-0", Namespace: "shop"},
		Containers: []v1beta1.ContainerMetrics{
			{Name: "web", Usage: v1.ResourceList{v1.ResourceCPU: resource.MustParse("250m"), v1.ResourceMemory: resource.MustParse("32Mi")}},
			{Name: "sidecar", Usage: v1.ResourceList{v1.ResourceCPU: resource.MustParse("10m"), v1.ResourceMemory: resource.MustParse("96Mi")}},
		},
	}

	clientset := fake.NewClientset(pod, systemPod)
	metricset := metricsfake.NewSimpleClientset()
	if err := metricset.Tracker().Create(podMetricsResource, metrics, metrics.Namespace); err != nil {
		t.Fatalf("seeding pod metrics: %v", err)
	}

	var metricsGets atomic.Int64
	metricset.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		metricsGets.Add(1)
		get := action.(k8stesting.GetAction)
		if get.GetName() != "web-0" || get.GetNamespace() != "shop" {
			t.Errorf("metrics requested for %s/%s, want shop/web-0", get.GetNamespace(), get.GetName())
		}
		return false, nil, nil
	})

	got, err := GetPodInfo(clientset, metricset)
	if err != nil {
		t.Fatalf("GetPodInfo returned an error: %v", err)
	}

	if calls := metricsGets.Load(); calls != 1 {
		t.Errorf("metrics fetched %d times for a two container pod, want 1", calls)
	}

	if len(got) != 2 {
		t.Fatalf("got %d entries, want one per container of the shop pod: %+v", len(got), got)
	}
	for _, info := range got {
		if info.Namespace == "kube-system" {
			t.Errorf("kube-system pod was not skipped: %+v", info)
		}
	}

	web, sidecar := got[0], got[1]
	if web.Container != "web" || sidecar.Container != "sidecar" {
		t.Fatalf("containers named %q and %q, want web and sidecar", web.Container, sidecar.Container)
	}
	if web.Name != "web-0" || web.Namespace != "shop" {
		t.Errorf("got pod %q in namespace %q, want web-0 in shop", web.Name, web.Namespace)
	}
	if web.Usage.Name != "web" || web.Usage.CPU != 250 || web.Usage.Memory != 32 {
		t.Errorf("web usage is %+v, want the web container's own usage", web.Usage)
	}
	if sidecar.Usage.Name != "sidecar" || sidecar.Usage.CPU != 10 || sidecar.Usage.Memory != 96 {
		t.Errorf("sidecar usage is %+v, want the sidecar container's own usage", sidecar.Usage)
	}
	if _, exists := web.Resources.Limits[v1.ResourceCPU]; !exists {
		t.Errorf("web carries limits %+v, want the cpu limit from its own container spec", web.Resources.Limits)
	}
	if _, exists := sidecar.Resources.Limits[v1.ResourceMemory]; !exists {
		t.Errorf("sidecar carries limits %+v, want the memory limit from its own container spec", sidecar.Resources.Limits)
	}
}

func TestGetPodInfoSkipsPodsWithoutMetrics(t *testing.T) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-0", Namespace: "shop"},
		Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "web"}}},
	}

	got, err := GetPodInfo(fake.NewClientset(pod), metricsfake.NewSimpleClientset())
	if err != nil {
		t.Fatalf("GetPodInfo returned an error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d entries, want none when metrics-server has nothing for the pod: %+v", len(got), got)
	}
}

func TestGetPodInfoReturnsListErrors(t *testing.T) {
	clientset := fake.NewClientset()
	clientset.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errListFailed
	})

	if _, err := GetPodInfo(clientset, metricsfake.NewSimpleClientset()); err == nil {
		t.Fatal("GetPodInfo returned no error when listing pods failed")
	}
}
