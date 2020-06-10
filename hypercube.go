package weighting

import (
	"fmt"
	"math"
)

// HyperCube is an implementation of the Weight interface
type HyperCube struct {
	Dimensions map[string]*Dimension
	Stats      *Stats
	Initspace  []float64
	Workspace  []float64
	DimValues  map[string]map[string]map[int]bool
	WorkRows   map[string]map[string]string
	Options    *Options
}

// Weight will register all  responses at once
func (hc *HyperCube) Weight(responses []*Response) (*Result, error) {
	for _, res := range responses {
		hc.analyzeDimensionsFromResponse(res)
	}
	hc.configureBlockSize()

	var err error
	for _, res := range responses {
		err = hc.addRow(res)
		if err != nil {
			return nil, err
		}
	}

	hc.rakeDimensions()
	return &Result{Weights: hc.getWeights(), Stats: hc.getStats()}, nil
}

func (hc *HyperCube) analyzeDimensionsFromResponse(r *Response) {
	for col, val := range r.Values {
		dim, found := hc.Dimensions[col]
		if !found {
			continue
		}

		if dim.Values == nil {
			dim.Values = make(map[string]*DimensionValue)
		}

		_, found = dim.Values[val]
		if !found {
			dim.Values[val] = NewDimensionValue(dim.Size)
			dim.Size++
		}
	}
}

func (hc *HyperCube) configureBlockSize() error {
	blockSize := int(1)
	for _, col := range hc.Options.Columns {
		dim := hc.Dimensions[col]
		dim.BlockSize = blockSize
		blockSize = dim.BlockSize * dim.Size
	}
	hc.Workspace = make([]float64, blockSize)
	hc.WorkRows = make(map[string]map[string]string)
	return nil
}

func (hc *HyperCube) addRow(r *Response) error {
	var isWork bool
	switch r.Values[hc.Options.GroupColumn] {
	case hc.Options.GoalGroupValue: // Exposed
		isWork = false
	case hc.Options.WorkGroupValue: // Control
		isWork = true
	default:
		return fmt.Errorf("invalid group value %s", r.Values[hc.Options.GroupColumn])
	}

	var index int
	for _, col := range hc.Options.Columns {
		val := hc.v(r.Values[col])
		dval, found := hc.Dimensions[col].Values[val]
		if !found {
			return fmt.Errorf("invalid value %s for column %s", val, col)
		}

		if isWork {
			dval.Initial++
			dval.WorkSum += 1.0
			hc.Dimensions[col].Initial++
			index += dval.Index * hc.Dimensions[col].BlockSize
		} else {
			dval.GoalSum += 1.0
		}
	}

	if isWork {
		hc.Stats.WorkRows++
		hc.Workspace[index] += 1.0
		hc.WorkRows[r.RespondentID] = make(map[string]string)
		for _, col := range hc.Options.Columns {
			_, found := hc.DimValues[col]
			if !found {
				hc.DimValues[col] = make(map[string]map[int]bool)
			}

			val := hc.v(r.Values[col])
			if hc.DimValues[col][val] == nil {
				hc.DimValues[col][val] = make(map[int]bool)
			}
			hc.DimValues[col][val][index] = true
			hc.WorkRows[r.RespondentID][col] = hc.v(r.Values[col])
		}
	} else {
		hc.Stats.GoalRows++
	}
	return nil
}

func (hc *HyperCube) rakeDimensions() {
	hc.Initspace = make([]float64, len(hc.Workspace))
	copy(hc.Initspace, hc.Workspace)

	iter := 0
	for iter < hc.Options.MaxIterations {
		dimAdjust := 1.0
		rmse := 0.0
		rmsn := 0
		hc.Stats.MinWeight = defaultWeight
		hc.Stats.MaxWeight = defaultWeight

		for _, c := range hc.Options.Columns {
			totalGoalSum := 0.0
			totalWorkSum := 0.0

			for k, data := range hc.Dimensions[c].Values {
				dv, found := hc.DimValues[c][k]
				if !found {
					continue
				}

				newWorkSum := 0.0
				for dvIndex := range dv {
					newWorkSum += hc.Workspace[dvIndex]
				}

				data.WorkSum = newWorkSum
				data.Weight, _ = hc.curbWeight(data.GoalSum / data.WorkSum)

				totalWorkSum += data.WorkSum
				totalGoalSum += data.GoalSum
			}

			hc.Dimensions[c].GoalSum = totalGoalSum
			hc.Dimensions[c].WorkSum = totalWorkSum

			dimAdjust *= hc.Dimensions[c].Adjustment()

			for k, data := range hc.Dimensions[c].Values {
				dv, found := hc.DimValues[c][k]
				if !found {
					continue
				}

				newWorkSum := 0.0
				for dvIndex := range dv {
					if hc.Workspace[dvIndex] < 0.00001 {
						hc.Workspace[dvIndex] = 0.00001
					}
					hc.Workspace[dvIndex] *= data.Weight
					newWorkSum += hc.Workspace[dvIndex]
				}
				data.WorkSum = newWorkSum / dimAdjust
				data.Weight, _ = hc.curbWeight(data.GoalSum / data.WorkSum)

				d2 := math.Pow(data.GoalSum-data.WorkSum, 2.0)
				data.Rmse += d2
				data.Rmsn++

				hc.Dimensions[c].Rmse += d2
				hc.Dimensions[c].Rmsn++

				rmse += d2
				rmsn++
			}
		}

		iter++
		hc.Stats.Iterations++
		hc.Stats.Rmse = math.Sqrt(rmse / float64(rmsn))
		if hc.Stats.Rmse <= hc.Options.RootMeanSquareError {
			break
		}
	}
	hc.setEffectiveBaseSize()
}

func (hc *HyperCube) getStats() *Stats {
	return hc.Stats
}

func (hc *HyperCube) getWeights() map[string]float64 {
	results := make(map[string]float64)
	for rid := range hc.WorkRows {
		_, _, weight := hc.weight(rid)
		results[rid] = weight
	}
	return results
}

func (hc *HyperCube) setEffectiveBaseSize() {
	for _, col := range hc.Options.Columns {
		hc.Dimensions[col].WorkSum = 0
		for _, dv := range hc.Dimensions[col].Values {
			dv.WorkSum = 0.0
		}
	}

	var totalCount, curbedCount int
	var totalSum, totalSum2 float64
	for rid, target := range hc.WorkRows {
		weighted, curbed, weight := hc.weight(rid)
		for _, col := range hc.Options.Columns {
			value := target[col]
			hc.Dimensions[col].WorkSum += weight
			hc.Dimensions[col].Values[value].WorkSum += weight
		}

		if weighted {
			if curbed {
				curbedCount++
			}

			totalSum += weight
			totalSum2 += math.Pow(weight, 2.0)
			totalCount++
		}
	}

	grandSum := totalSum + float64(hc.Stats.GoalRows)
	grandSum2 := totalSum2 + float64(hc.Stats.GoalRows)
	grandCount := totalCount + hc.Stats.GoalRows

	hc.Stats.AverageWeight = totalSum / float64(totalCount)
	hc.Stats.DesignEffect = float64(grandCount) * (grandSum2 / (math.Pow(grandSum, 2.0)))
	hc.Stats.EffectiveBaseSize = math.Pow(totalSum, 2.0) / totalSum2
	hc.Stats.Curbed = (100.0 * float64(curbedCount)) / float64(totalCount)
}

func (hc *HyperCube) weight(responseID string) (weighted, curbed bool, weight float64) {
	target, found := hc.WorkRows[responseID]
	if !found || target == nil {
		return false, false, defaultWeight
	}

	index := hc.rowIndex(target)
	weight = (hc.Workspace[index] * float64(hc.Stats.WorkRows)) /
		(hc.Initspace[index] * float64(hc.Stats.GoalRows)) //	/ hc.Stats.AverageWeight

	if weight < hc.Stats.MinWeight {
		hc.Stats.MinWeight = weight
	}
	if weight > hc.Stats.MaxWeight {
		hc.Stats.MaxWeight = weight
	}

	weight, curbed = hc.curbWeight(weight)
	return true, curbed, weight
}

func (hc *HyperCube) rowIndex(res map[string]string) int {
	var index int
	for _, c := range hc.Options.Columns {
		val := hc.v(res[c])
		index += hc.Dimensions[c].Values[val].Index * hc.Dimensions[c].BlockSize
	}
	return index
}

func (hc *HyperCube) curbWeight(weight float64) (float64, bool) {
	if weight < hc.Options.LowerWeightCap {
		return hc.Options.LowerWeightCap, true
	}

	if weight > hc.Options.UpperWeightCap {
		return hc.Options.UpperWeightCap, true
	}
	return weight, false
}

func (hc *HyperCube) v(value string) string {
	// TODO: Placeholder for value translations
	return value
}

// NewGroupedWeighter returns an instance of the HyperCube implementation
func NewGroupedWeighter(ops *Options) *HyperCube {
	hc := &HyperCube{Options: ops}
	hc.Dimensions = make(map[string]*Dimension)
	hc.DimValues = make(map[string]map[string]map[int]bool)
	hc.Stats = &Stats{
		AverageWeight: defaultWeight,
		MinWeight:     defaultWeight,
		MaxWeight:     defaultWeight,
	}
	for _, c := range ops.Columns {
		hc.Dimensions[c] = NewDimension()
	}
	return hc
}
