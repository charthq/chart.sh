package server

import (
	"time"

	chartserverv1beta1 "github.com/charthq/chartserver/pkg/apis/chartserver/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
)

func (h *ChartServer) refreshCache() error {
	namespaces, err := h.client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list namespaces")
	}

	foundCharts := []chartserverv1beta1.Chart{}
	foundChartVersions := []chartserverv1beta1.ChartVersion{}

	for _, namespace := range namespaces.Items {
		charts, err := h.chartserverClient.Charts(namespace.Name).List(metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to list charts in namespace")
		}

		for _, chart := range charts.Items {
			foundCharts = append(foundCharts, chart)
		}

		chartVersions, err := h.chartserverClient.ChartVersions(namespace.Name).List(metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to list chart versions in namespace")
		}

		for _, chartVersion := range chartVersions.Items {
			foundChartVersions = append(foundChartVersions, chartVersion)
		}
	}

	chartCache := map[string]repo.ChartVersions{}

	for _, foundChart := range foundCharts {
		indexChartVersions := repo.ChartVersions{}

		for _, foundChartVersion := range foundChartVersions {
			if foundChartVersion.Spec.Name == foundChart.Spec.Name {
				// https://github.com/helm/helm/blob/master/pkg/proto/hapi/chart/metadata.pb.go#L105

				metadata := chart.Metadata{
					Name: foundChartVersion.Spec.Name,
					Home: foundChartVersion.Spec.Home,
				}
				indexChartVersion := repo.ChartVersion{
					Metadata: &metadata,
				}

				indexChartVersions = append(indexChartVersions, &indexChartVersion)
			}
		}

		if indexChartVersions.Len() > 0 {
			chartCache[foundChart.Spec.Name] = indexChartVersions
		}
	}

	h.chartCache = chartCache
	h.cacheGeneratedAt = time.Now()

	return nil
}
