package main

import (
	"reflect"
	"testing"
)

func Test_getResolvedConfig(t *testing.T) {
	type args struct {
		topic                          string
		topicConfig                    map[string]string
		newConfig                      map[string]string
		minValueOverrideForTopicConfig map[string]int64
	}
	tests := []struct {
		name            string
		args            args
		wantResolved    map[string]string
		wantNeedsUpdate bool
		shouldPanic     bool
	}{
		{
			name: "needsUpdate",
			args: args{
				topic:                          "t1",
				topicConfig:                    map[string]string{"a": "textval1", "b": "textval2", "c": "1000", "d": "2000", "e": "1000"},
				newConfig:                      map[string]string{"x": "textval3", "b": "textval2", "c": "500", "d": "1000"},
				minValueOverrideForTopicConfig: map[string]int64{"d": 3000, "f": 5000},
			},
			wantResolved:    map[string]string{"x": "textval3", "b": "textval2", "c": "500", "d": "3000", "f": "5000"},
			wantNeedsUpdate: true,
			shouldPanic:     false,
		},
		{
			name: "doesntNeedUpdate_newConfigSameOrLowerButExistingEqualsMinOverride",
			args: args{
				topic:                          "t2",
				topicConfig:                    map[string]string{"x": "1000", "y": "2000"},
				newConfig:                      map[string]string{"x": "1000", "y": "1000"},
				minValueOverrideForTopicConfig: map[string]int64{"x": 1000, "y": 2000},
			},
			wantResolved:    map[string]string{"x": "1000", "y": "2000"},
			wantNeedsUpdate: false,
			shouldPanic:     false,
		},
		{
			name: "doesntNeedUpdate_compactDeleteVariants",
			args: args{
				topic:                          "t3",
				topicConfig:                    map[string]string{"1": "[compact,delete]", "2": "[delete,compact]", "3": "delete,compact", "4": "compact,delete"},
				newConfig:                      map[string]string{"2": "[compact,delete]", "3": "[delete,compact]", "4": "delete,compact", "1": "compact,delete"},
				minValueOverrideForTopicConfig: map[string]int64{},
			},
			wantResolved:    map[string]string{"1": "compact,delete", "2": "compact,delete", "3": "compact,delete", "4": "compact,delete"},
			wantNeedsUpdate: false,
			shouldPanic:     false,
		},
		{
			name: "shouldPanic_newConfigMinOverrideIntParsing",
			args: args{
				topic:                          "t4",
				topicConfig:                    map[string]string{"d": "2000"},
				newConfig:                      map[string]string{"d": "blah"},
				minValueOverrideForTopicConfig: map[string]int64{"d": 2000},
			},
			wantResolved:    map[string]string{},
			wantNeedsUpdate: false,
			shouldPanic:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("The code did not panic for %s test but expected to panic", tt.name)
					}
				}()
			}
			gotResolvedConfig, gotNeedsUpdate := getResolvedConfig(tt.args.topic, tt.args.topicConfig, tt.args.newConfig, tt.args.minValueOverrideForTopicConfig)
			if !reflect.DeepEqual(gotResolvedConfig, tt.wantResolved) {
				t.Errorf("getResolvedConfig() gotResolvedConfig = %v, wantResolved %v", gotResolvedConfig, tt.wantResolved)
			}
			if gotNeedsUpdate != tt.wantNeedsUpdate {
				t.Errorf("getResolvedConfig() gotNeedsUpdate = %v, wantResolved %v", gotNeedsUpdate, tt.wantNeedsUpdate)
			}
		})
	}
}
