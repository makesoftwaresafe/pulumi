// Copyright 2016-2018, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type TestStruct struct {
	Foo map[string]string
	Bar string
}

// TestConfig tests the basic config wrapper.
func TestConfig(t *testing.T) {
	t.Parallel()

	ctx, err := pulumi.NewContext(context.Background(), pulumi.RunInfo{
		Config: map[string]string{
			"testpkg:sss":    "a string value",
			"testpkg:bbb":    "true",
			"testpkg:intint": "42",
			"testpkg:badint": "4d2",
			"testpkg:fpfpfp": "99.963",
			"testpkg:obj": `
				{
					"foo": {
						"a": "1",
						"b": "2"
					},
					"bar": "abc"
				}
			`,
			"testpkg:malobj": "not_a_struct",
		},
	})
	require.NoError(t, err)

	cfg := New(ctx, "testpkg")

	var testStruct TestStruct
	var emptyTestStruct TestStruct

	fooMap := make(map[string]string)
	fooMap["a"] = "1"
	fooMap["b"] = "2"
	expectedTestStruct := TestStruct{
		Foo: fooMap,
		Bar: "abc",
	}

	// Test basic keys.
	assert.Equal(t, "testpkg:sss", cfg.fullKey("sss"))

	// Test Get, which returns a default value for missing entries rather than failing.
	assert.Equal(t, "a string value", cfg.Get("sss"))
	assert.Equal(t, true, cfg.GetBool("bbb"))
	assert.Equal(t, 42, cfg.GetInt("intint"))
	assert.Equal(t, 99.963, cfg.GetFloat64("fpfpfp"))
	assert.Equal(t, "", cfg.Get("missing"))
	// missing key GetObj
	err = cfg.GetObject("missing", &testStruct)
	assert.Equal(t, emptyTestStruct, testStruct)
	require.NoError(t, err)
	testStruct = TestStruct{}
	// malformed key GetObj
	err = cfg.GetObject("malobj", &testStruct)
	assert.Equal(t, emptyTestStruct, testStruct)
	assert.ErrorContains(t, err, "invalid character 'o' in literal null (expecting 'u')")
	testStruct = TestStruct{}
	// GetObj
	err = cfg.GetObject("obj", &testStruct)
	assert.Equal(t, expectedTestStruct, testStruct)
	require.NoError(t, err)
	testStruct = TestStruct{}

	// Test Require, which panics for missing entries.
	assert.Equal(t, "a string value", cfg.Require("sss"))
	assert.Equal(t, true, cfg.RequireBool("bbb"))
	assert.Equal(t, 42, cfg.RequireInt("intint"))
	assert.PanicsWithError(t,
		"unable to parse required configuration variable"+
			" 'testpkg:badint'; unable to cast \"4d2\" of type string to int",
		func() { cfg.RequireInt("badint") })
	assert.Equal(t, 99.963, cfg.RequireFloat64("fpfpfp"))
	cfg.RequireObject("obj", &testStruct)
	assert.Equal(t, expectedTestStruct, testStruct)
	testStruct = TestStruct{}
	// GetObj panics if value is malformed
	willPanic := func() { cfg.RequireObject("malobj", &testStruct) }
	assert.Panics(t, willPanic)
	testStruct = TestStruct{}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected malformed value for RequireObject to panic")
			}
		}()
		cfg.RequireObject("malobj", &testStruct)
	}()
	testStruct = TestStruct{}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected missing key for Require to panic")
			}
		}()
		_ = cfg.Require("missing")
	}()

	// Test Try, which returns an error for missing or invalid entries.
	k1, err := cfg.Try("sss")
	require.NoError(t, err)
	assert.Equal(t, "a string value", k1)
	k2, err := cfg.TryBool("bbb")
	require.NoError(t, err)
	assert.Equal(t, true, k2)
	k3, err := cfg.TryInt("intint")
	require.NoError(t, err)
	assert.Equal(t, 42, k3)
	invalidInt, err := cfg.TryInt("badint")
	assert.ErrorContains(t, err, "unable to cast \"4d2\" of type string to int")
	assert.Zero(t, invalidInt)
	k4, err := cfg.TryFloat64("fpfpfp")
	require.NoError(t, err)
	assert.Equal(t, 99.963, k4)
	// happy path TryObject
	err = cfg.TryObject("obj", &testStruct)
	require.NoError(t, err)
	assert.Equal(t, expectedTestStruct, testStruct)
	testStruct = TestStruct{}
	// missing TryObject
	err = cfg.TryObject("missing", &testStruct)
	assert.EqualError(t, err, "missing required configuration variable 'testpkg:missing'; run `pulumi config` to set")
	assert.Equal(t, emptyTestStruct, testStruct)
	assert.True(t, errors.Is(err, ErrMissingVar))
	testStruct = TestStruct{}
	// malformed TryObject
	err = cfg.TryObject("malobj", &testStruct)
	assert.EqualError(t, err, "invalid character 'o' in literal null (expecting 'u')")
	assert.Equal(t, emptyTestStruct, testStruct)
	assert.False(t, errors.Is(err, ErrMissingVar))
	testStruct = TestStruct{}
	_, err = cfg.Try("missing")
	assert.EqualError(t, err,
		"missing required configuration variable 'testpkg:missing'; run `pulumi config` to set")
	assert.True(t, errors.Is(err, ErrMissingVar))
}

func TestSecretConfig(t *testing.T) {
	t.Parallel()

	ctx, err := pulumi.NewContext(context.Background(), pulumi.RunInfo{
		Config: map[string]string{
			"testpkg:sss":    "a string value",
			"testpkg:bbb":    "true",
			"testpkg:intint": "42",
			"testpkg:fpfpfp": "99.963",
			"testpkg:obj": `
				{
					"foo": {
						"a": "1",
						"b": "2"
					},
					"bar": "abc"
				}
			`,
			"testpkg:malobj": "not_a_struct",
		},
	})
	require.NoError(t, err)

	cfg := New(ctx, "testpkg")

	fooMap := make(map[string]string)
	fooMap["a"] = "1"
	fooMap["b"] = "2"
	expectedTestStruct := TestStruct{
		Foo: fooMap,
		Bar: "abc",
	}

	s1, err := cfg.TrySecret("sss")
	s2 := cfg.RequireSecret("sss")
	s3 := cfg.GetSecret("sss")
	require.NoError(t, err)

	errChan := make(chan error)
	result := make(chan string)

	pulumi.All(s1, s2, s3).ApplyT(func(v []interface{}) ([]interface{}, error) {
		for _, val := range v {
			if val == "a string value" {
				result <- val.(string)
			} else {
				errChan <- fmt.Errorf("invalid result: %v", val)
			}
		}
		return v, nil
	})

	for i := 0; i < 3; i++ {
		select {
		case err = <-errChan:
			require.NoError(t, err)
			break
		case r := <-result:
			assert.Equal(t, "a string value", r)
			break
		}
	}

	errChan = make(chan error)
	objResult := make(chan TestStruct)

	testStruct4 := TestStruct{}
	testStruct5 := TestStruct{}
	testStruct6 := TestStruct{}

	s4, err := cfg.TrySecretObject("obj", &testStruct4)
	require.NoError(t, err)
	s5 := cfg.RequireSecretObject("obj", &testStruct5)
	s6, err := cfg.GetSecretObject("obj", &testStruct6)
	require.NoError(t, err)

	pulumi.All(s4, s5, s6).ApplyT(func(v []interface{}) ([]interface{}, error) {
		for _, val := range v {
			ts := val.(*TestStruct)
			if reflect.DeepEqual(expectedTestStruct, *ts) {
				objResult <- *ts
			} else {
				errChan <- fmt.Errorf("invalid result: %v", val)
			}
		}
		return v, nil
	})

	for i := 0; i < 3; i++ {
		select {
		case err = <-errChan:
			require.NoError(t, err)
			break
		case o := <-objResult:
			assert.Equal(t, expectedTestStruct, o)
			break
		}
	}

	s7, err := cfg.TrySecretBool("bbb")
	s8 := cfg.RequireSecretBool("bbb")
	s9 := cfg.GetSecretBool("bbb")
	require.NoError(t, err)

	errChan = make(chan error)
	resultBool := make(chan bool)

	pulumi.All(s7, s8, s9).ApplyT(func(v []interface{}) ([]interface{}, error) {
		for _, val := range v {
			if val == true {
				resultBool <- val.(bool)
			} else {
				errChan <- fmt.Errorf("invalid result: %v", val)
			}
		}
		return v, nil
	})

	for i := 0; i < 3; i++ {
		select {
		case err = <-errChan:
			require.NoError(t, err)
			break
		case r := <-resultBool:
			assert.Equal(t, true, r)
			break
		}
	}

	s10, err := cfg.TrySecretInt("intint")
	s11 := cfg.RequireSecretInt("intint")
	s12 := cfg.GetSecretInt("intint")
	require.NoError(t, err)

	errChan = make(chan error)
	resultInt := make(chan int)

	pulumi.All(s10, s11, s12).ApplyT(func(v []interface{}) ([]interface{}, error) {
		for _, val := range v {
			if val == 42 {
				resultInt <- val.(int)
			} else {
				errChan <- fmt.Errorf("invalid result: %v", val)
			}
		}
		return v, nil
	})

	for i := 0; i < 3; i++ {
		select {
		case err = <-errChan:
			require.NoError(t, err)
			break
		case r := <-resultInt:
			assert.Equal(t, 42, r)
			break
		}
	}
}
