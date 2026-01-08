// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/luxfi/metric"
	dto "github.com/prometheus/client_model/go"
)

var (
	_ MultiGatherer = (*prefixGatherer)(nil)

	errDuplicateGatherer = errors.New("attempt to register duplicate gatherer")
)

// NewLabelGatherer returns a new MultiGatherer that merges metrics by adding a
// new label.
func NewLabelGatherer(labelName string) MultiGatherer {
	return &labelGatherer{
		labelName: labelName,
	}
}

type labelGatherer struct {
	multiGatherer

	labelName string
}

func (g *labelGatherer) Gather() ([]*dto.MetricFamily, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	// Map to merge metrics by family name
	familyMap := make(map[string]*dto.MetricFamily)
	var gathererError error

	for _, gatherer := range g.gatherers {
		families, err := gatherer.Gather()
		// Store error but continue gathering
		if err != nil && gathererError == nil {
			gathererError = err
		}

		for _, family := range families {
			name := *family.Name
			if existingFamily, ok := familyMap[name]; ok {
				// Check for label conflicts - if any metric pair has all the same labels,
				// that's a conflict
				hasConflict := false
				for _, newMetric := range family.Metric {
					for _, existingMetric := range existingFamily.Metric {
						if labelsEqual(newMetric.Label, existingMetric.Label) {
							gathererError = fmt.Errorf("duplicate metrics in family %q", name)
							hasConflict = true
							break
						}
					}
					if hasConflict {
						break
					}
				}
				// Only merge if no conflict
				if !hasConflict {
					existingFamily.Metric = append(existingFamily.Metric, family.Metric...)
				}
			} else {
				// Add new family
				familyMap[name] = family
			}
		}
	}

	// Convert map to sorted slice
	var result []*dto.MetricFamily
	for _, family := range familyMap {
		// Sort metrics within each family by label values
		sort.Slice(family.Metric, func(i, j int) bool {
			return compareMetrics(family.Metric[i], family.Metric[j]) < 0
		})
		result = append(result, family)
	}

	// Sort families by name
	sort.Slice(result, func(i, j int) bool {
		return *result[i].Name < *result[j].Name
	})

	return result, gathererError
}

func labelsEqual(labels1, labels2 []*dto.LabelPair) bool {
	if len(labels1) != len(labels2) {
		return false
	}
	// Create a map of labels1
	labelMap := make(map[string]string)
	for _, label := range labels1 {
		labelMap[*label.Name] = *label.Value
	}
	// Check if labels2 matches
	for _, label := range labels2 {
		if val, ok := labelMap[*label.Name]; !ok || val != *label.Value {
			return false
		}
	}
	return true
}

func compareMetrics(m1, m2 *dto.Metric) int {
	// Compare metrics by their label values
	for i := 0; i < len(m1.Label) && i < len(m2.Label); i++ {
		if *m1.Label[i].Name != *m2.Label[i].Name {
			if *m1.Label[i].Name < *m2.Label[i].Name {
				return -1
			}
			return 1
		}
		if *m1.Label[i].Value != *m2.Label[i].Value {
			if *m1.Label[i].Value < *m2.Label[i].Value {
				return -1
			}
			return 1
		}
	}
	if len(m1.Label) < len(m2.Label) {
		return -1
	}
	if len(m1.Label) > len(m2.Label) {
		return 1
	}
	return 0
}

func (g *labelGatherer) Register(labelValue string, gatherer metric.Gatherer) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if slices.Contains(g.names, labelValue) {
		return fmt.Errorf("%w: for %q with label %q",
			errDuplicateGatherer,
			g.labelName,
			labelValue,
		)
	}

	g.register(
		labelValue,
		&labeledGatherer{
			labelName:  g.labelName,
			labelValue: labelValue,
			gatherer:   gatherer,
		},
	)
	return nil
}

type labeledGatherer struct {
	labelName  string
	labelValue string
	gatherer   metric.Gatherer
}

func (g *labeledGatherer) Gather() ([]*dto.MetricFamily, error) {
	// Gather returns partially filled metrics in the case of an error. So, it
	// is expected to still return the metrics in the case an error is returned.
	metricFamilies, err := g.gatherer.Gather()
	var labelError error

	for _, metricFamily := range metricFamilies {
		var validMetrics []*dto.Metric
		for _, metric := range metricFamily.Metric {
			// Check if the label already exists
			hasConflict := false
			for _, existingLabel := range metric.Label {
				if *existingLabel.Name == g.labelName {
					// Label already exists, this is an error
					if labelError == nil {
						labelError = fmt.Errorf("label %q is already present in metric %q", g.labelName, *metricFamily.Name)
					}
					hasConflict = true
					break
				}
			}

			if !hasConflict {
				metric.Label = append(metric.Label, &dto.LabelPair{
					Name:  &g.labelName,
					Value: &g.labelValue,
				})
				// Sort labels by name to ensure consistent ordering
				sort.Slice(metric.Label, func(i, j int) bool {
					return *metric.Label[i].Name < *metric.Label[j].Name
				})
				validMetrics = append(validMetrics, metric)
			}
			// If there's a conflict, skip this metric entirely
		}
		// Update the metric family with only valid metrics
		metricFamily.Metric = validMetrics
	}

	// Return the original error if present, otherwise the label error
	if err != nil {
		return metricFamilies, err
	}
	return metricFamilies, labelError
}
