# Go-rim-weighting

GoLang library for RIM (Random Iterative Method) Weighting of Survey data.

Currently, this package only implements the "Grouped" mode - ideal for when your population is clearly grouped into two separate groups, a work (control) group and a goal (exposed) group, the object of weighting is to make the work group conform to the goal group.


NOTE: This is a Golang implementation of a Ruby Gem internally used in Dynata - this was written to aide with the transition to writing more code in Golang. As we make progress in this direction, we plan to add more weighting methods to this package and as a result the API may change.

### Weighter

This package exposes a `Weighter` interface that all implementations will conform to.

```go
type Weighter interface {
	Weight(responses []*Response) (*Result, error)
}
```

### Grouped Mode

In grouped mode, we are weighting one group (the "work" group) to conform to another group (the "goal" group). When in grouped mode, the following parameters apply:

```go
    options := &weighting.Options{
        Columns: []string{"d1", "d2", "d3"},
        GroupColumn: "goal_or_group",
        GoalGroupValue: "goal_group",
        WorkGroupValue: "work_group",
    }

    var responses []*weighting.Response{
        &weighting.Response{
            RespondentID: "r1",
            Values: map[string]string{
                "d1": "1",
                "d2": "2",
                "d3": "3",
            },
        },
        &weighting.Response{
            RespondentID: "r2",
            Values: map[string]string{
                "d1": "5",
                "d2": "6",
                "d3": "7",
            },
        },
        ...
    }

    weighter := weighting.NewGroupedWeighter(options)
    res, err := weighter.Weight(responses)
    if err != nil {
        fmt.Println(err)
        return
    }

    fmt.Println(res.Weights) //map[string]float64
    //map["1000"]: 0.7252778761433506
    //key == Respondent ID (or response id) and value = Weight

    fmt.Println(res.Stats) 
    /*
    Example: 
    &weighting.Stats{GoalRows:487, WorkRows:513, Iterations:2, Rmse:5.1609399352421446e-14, AverageWeight:1.000421552877844, DesignEffect:1.1269921846722624, EffectiveBaseSize:411.2400998141462, Curbed:1.1695906432748537, MinWeight:0.2098982773153223, MaxWeight:3.4068774426214232}*/
```

The `GroupColumn` and `Columns` parameters are required. The `GroupColumn` values identify which member is in which group. By default, the `GroupColumn` is 1 for goal and 2 for work. You can customize these values using the `GoalGroupValue` and `WorkGroupValue` parameters. The `Columns` parameter identifies the member attributes (or questions) you wish to weight on.


The returned respondents are in random order and **_only contain work group members_** (in grouped mode). The goal group members can be assumed to have a weighting of 1.0 and thus need not be returned in the output payload.

### Usage

Please see the tests for example of usage.


### To Dos

* Add other weighting methods (Flatspace, nested ...)
* Clean up and add more comments
* Add more testing coverage


### Maintainers

* Saad (saadullah.saeed@dynata.com)