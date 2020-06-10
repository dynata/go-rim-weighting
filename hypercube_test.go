package weighting

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestHypercube_Weight(t *testing.T) {
	options := &Options{
		Columns:             []string{"D1", "D2", "D3", "D4", "D5", "W1", "W2", "W3"},
		GroupColumn:         "hType",
		GoalGroupValue:      "1",
		WorkGroupValue:      "2",
		MaxIterations:       50,
		UpperWeightCap:      5.0,
		LowerWeightCap:      0.3,
		RootMeanSquareError: 1.0e-6,
	}

	responses, err := readResponsesFromCSV("data.csv")
	if err != nil {
		t.Error(err)
		return
	}

	weighter := NewGroupedWeighter(options)
	result, err := weighter.Weight(responses)
	if err != nil {
		t.Error(err)
		return
	}

	jsonContent, err := ioutil.ReadFile("testdata/expected.json")
	if err != nil {
		t.Error(err)
		return
	}

	var weights map[string]float64
	err = json.Unmarshal(jsonContent, &weights)
	if err != nil {
		t.Error(err)
		return
	}

	for k, v := range weights {
		// upto 12 decimal places
		if fmt.Sprintf("%.12f", v) != fmt.Sprintf("%.12f", result.Weights[k]) {
			fmt.Println(k, v, result.Weights[k])
			t.Fail()
		}
	}

}

func readResponsesFromCSV(file string) ([]*Response, error) {
	csvfile, err := os.Open(fmt.Sprintf("testdata/%s", file))
	if err != nil {
		return nil, err
	}

	var responses []*Response
	r := csv.NewReader(csvfile)
	var header []string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header == nil {
			header = record
			continue
		}

		res := &Response{
			RespondentID: record[0],
			Values:       make(map[string]string),
		}
		for i, val := range record {
			if i == 0 {
				continue
			}
			res.Values[header[i]] = val
		}
		responses = append(responses, res)
	}
	return responses, nil
}
