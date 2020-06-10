package weighting

import (
	"encoding/json"
)

const (
	defaultWeight              = 1.0
	defaultAdjustment          = 10.0
	defaultWeightColumn        = "weight"
	defaultMaxIterations       = 50
	defaultUpperWeightCap      = 5.0
	defaultLowerWeightCap      = 0.3
	defaultRootMeanSquareError = 1.0e-6
)

// Options represent the weighting Options
type Options struct {
	WeightColumn        string  `json:"weight_column,omitempty"`
	MaxIterations       int     `json:"max_iterations,omitempty"`
	UpperWeightCap      float64 `json:"upper_weight_cap,omitempty"`
	LowerWeightCap      float64 `json:"lower_weight_cap,omitempty"`
	RootMeanSquareError float64 `json:"root_mean_square_error,omitempty"`

	Columns        []string `json:"columns,omitempty"`
	GroupColumn    string   `json:"group_column,omitempty"`
	GoalGroupValue string   `json:"goal_group_value,omitempty"`
	WorkGroupValue string   `json:"work_group_value,omitempty"`
}

// NewDefaultOptions ...
func NewDefaultOptions() *Options {
	return &Options{
		WeightColumn:        defaultWeightColumn,
		MaxIterations:       defaultMaxIterations,
		UpperWeightCap:      defaultUpperWeightCap,
		LowerWeightCap:      defaultLowerWeightCap,
		RootMeanSquareError: defaultRootMeanSquareError,
	}
}

// NewOptionsFromJSONConfig ...
func NewOptionsFromJSONConfig(config json.RawMessage) (*Options, error) {
	ops := NewDefaultOptions()
	err := json.Unmarshal(config, &ops)
	return ops, err
}

// Response represents a single respondents entire response
type Response struct {
	RespondentID string
	Values       map[string]string
}

type Dimension struct {
	Size      int
	Initial   int
	BlockSize int
	GoalSum   float64
	WorkSum   float64
	Rmse      float64
	Rmsn      int
	Values    map[string]*DimensionValue
}

func (d *Dimension) Adjustment() float64 {
	if d.WorkSum > 0.0 {
		return d.GoalSum / d.WorkSum
	}
	return defaultAdjustment
}

func NewDimension() *Dimension {
	return &Dimension{}
}

type DimensionValue struct {
	Index   int
	Initial int
	GoalSum float64
	WorkSum float64
	Ratio   float64
	Rmse    float64
	Rmsn    int
	Weight  float64
}

func NewDimensionValue(index int) *DimensionValue {
	return &DimensionValue{
		Index:  index,
		Weight: defaultWeight,
	}
}

// Stats ...
type Stats struct {
	GoalRows          int     `json:"goalRows"`
	WorkRows          int     `json:"workRows"`
	Iterations        int     `json:"iterations"`
	Rmse              float64 `json:"rmse"`
	AverageWeight     float64 `json:"averageWeight"`
	DesignEffect      float64 `json:"designEffect"`
	EffectiveBaseSize float64 `json:"effectiveBaseSize"`
	Curbed            float64 `json:"curbed"`
	MinWeight         float64 `json:"minWeight"`
	MaxWeight         float64 `json:"maxWeight"`
	CsvData           string  `json:"csvData"`
}

// Result ...
type Result struct {
	Weights map[string]float64
	Stats   *Stats
}

// Weighter interface defines the contract for weighting implementations
type Weighter interface {
	Weight(responses []*Response) (*Result, error)
}

func contains(array []string, s string) bool {
	for _, a := range array {
		if a == s {
			return true
		}
	}
	return false
}
