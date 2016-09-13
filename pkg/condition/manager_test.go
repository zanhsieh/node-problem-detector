/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package condition

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/problemclient"
	"k8s.io/node-problem-detector/pkg/types"
	problemutil "k8s.io/node-problem-detector/pkg/util"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util"
)

func newTestManager() (*conditionManager, *problemclient.FakeProblemClient, *util.FakeClock) {
	fakeClient := problemclient.NewFakeProblemClient()
	fakeClock := util.NewFakeClock(time.Now())
	manager := NewConditionManager(fakeClient, fakeClock)
	return manager.(*conditionManager), fakeClient, fakeClock
}

func newTestCondition(condition string) types.Condition {
	return types.Condition{
		Type:       condition,
		Status:     true,
		Transition: time.Now(),
		Reason:     "TestReason",
		Message:    "test message",
	}
}

func TestCheckUpdates(t *testing.T) {
	m, _, _ := newTestManager()
	var c types.Condition
	for desc, test := range map[string]struct {
		condition string
		update    bool
	}{
		"Init condition needs update": {
			condition: "TestCondition",
			update:    true,
		},
		"Same condition doesn't need update": {
			// not set condition, the test will reuse the condition in last case.
			update: false,
		},
		"Same condition with different timestamp need update": {
			condition: "TestCondition",
			update:    true,
		},
		"New condition needs update": {
			condition: "TestConditionNew",
			update:    true,
		},
	} {
		if test.condition != "" {
			c = newTestCondition(test.condition)
		}
		m.UpdateCondition(c)
		assert.Equal(t, test.update, m.checkUpdates(), desc)
		assert.Equal(t, c, m.conditions[c.Type], desc)
	}
}

func TestSync(t *testing.T) {
	m, fakeClient, fakeClock := newTestManager()
	condition := newTestCondition("TestCondition")
	m.conditions = map[string]types.Condition{condition.Type: condition}
	m.sync()
	expected := []api.NodeCondition{problemutil.ConvertToAPICondition(condition)}
	assert.Nil(t, fakeClient.AssertConditions(expected), "Condition should be updated via client")
	assert.False(t, m.checkResync(), "Should not resync before timeout exceeds")
	fakeClock.Step(resyncPeriod)
	assert.True(t, m.checkResync(), "Should resync after timeout exceeds")
}
