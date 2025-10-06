package registry

import "github.com/prometheus/prometheus/model/labels"

func getLabelsFromValueCombo(labelValueCombo *LabelValueCombo) labels.Labels {
	lbls := labelValueCombo.getLabelPair()
	lb := make([]labels.Label, len(lbls.names))
	for i := range lbls.names {
		lb[i] = labels.Label{Name: lbls.names[i], Value: lbls.values[i]}
	}
	l := labels.New(lb...)
	return l
}
