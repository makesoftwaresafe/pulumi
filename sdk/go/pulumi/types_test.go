// Copyright 2016-2022, Pulumi Corporation.
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

//nolint:lll
package pulumi

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func await(out Output) (interface{}, bool, bool, []Resource, error) {
	return awaitWithContext(context.Background(), out)
}

func assertApplied(t *testing.T, out Output) {
	_, known, _, _, err := await(out)
	assert.True(t, known)
	assert.Nil(t, err)
}

func newIntOutput() IntOutput {
	return IntOutput{newOutputState(nil, reflect.TypeOf(42))}
}

func TestBasicOutputs(t *testing.T) {
	t.Parallel()

	// Just test basic resolve and reject functionality.
	{
		out, resolve, _ := NewOutput()
		go func() {
			resolve(42)
		}()
		v, known, secret, deps, err := await(out)
		assert.Nil(t, err)
		assert.True(t, known)
		assert.False(t, secret)
		assert.Nil(t, deps)
		assert.NotNil(t, v)
		assert.Equal(t, 42, v.(int))
	}
	{
		out, _, reject := NewOutput()
		go func() {
			reject(errors.New("boom"))
		}()
		v, _, _, _, err := await(out)
		assert.NotNil(t, err)
		assert.Nil(t, v)
	}
}

func TestArrayOutputs(t *testing.T) {
	t.Parallel()

	out := ArrayOutput{newOutputState(nil, reflect.TypeOf([]interface{}{}))}
	go func() {
		out.resolve([]interface{}{nil, 0, "x"}, true, false, nil)
	}()
	{
		assertApplied(t, out.ApplyT(func(arr []interface{}) (interface{}, error) {
			assert.NotNil(t, arr)
			if assert.Equal(t, 3, len(arr)) {
				assert.Equal(t, nil, arr[0])
				assert.Equal(t, 0, arr[1])
				assert.Equal(t, "x", arr[2])
			}
			return nil, nil
		}))
	}
}

func TestBoolOutputs(t *testing.T) {
	t.Parallel()

	out := BoolOutput{newOutputState(nil, reflect.TypeOf(false))}
	go func() {
		out.resolve(true, true, false, nil)
	}()
	{
		assertApplied(t, out.ApplyT(func(v bool) (interface{}, error) {
			assert.True(t, v)
			return nil, nil
		}))
	}
}

func TestMapOutputs(t *testing.T) {
	t.Parallel()

	out := MapOutput{newOutputState(nil, reflect.TypeOf(map[string]interface{}{}))}
	go func() {
		out.resolve(map[string]interface{}{
			"x": 1,
			"y": false,
			"z": "abc",
		}, true, false, nil)
	}()
	{
		assertApplied(t, out.ApplyT(func(v map[string]interface{}) (interface{}, error) {
			assert.NotNil(t, v)
			assert.Equal(t, 1, v["x"])
			assert.Equal(t, false, v["y"])
			assert.Equal(t, "abc", v["z"])
			return nil, nil
		}))
	}
}

func TestNumberOutputs(t *testing.T) {
	t.Parallel()

	out := Float64Output{newOutputState(nil, reflect.TypeOf(float64(0)))}
	go func() {
		out.resolve(42.345, true, false, nil)
	}()
	{
		assertApplied(t, out.ApplyT(func(v float64) (interface{}, error) {
			assert.Equal(t, 42.345, v)
			return nil, nil
		}))
	}
}

func TestStringOutputs(t *testing.T) {
	t.Parallel()

	out := StringOutput{newOutputState(nil, reflect.TypeOf(""))}
	go func() {
		out.resolve("a stringy output", true, false, nil)
	}()
	{
		assertApplied(t, out.ApplyT(func(v string) (interface{}, error) {
			assert.Equal(t, "a stringy output", v)
			return nil, nil
		}))
	}
}

func TestAliasedOutputs(t *testing.T) {
	t.Parallel()

	// Irrelevant for the tests, we're testing return type handling.
	initialOutput := String("").ToStringOutput()

	t.Run("Bool", func(t *testing.T) {
		t.Parallel()
		assertApplied(t, initialOutput.ApplyT(func(v interface{}) (Bool, error) {
			return Bool(false), nil
		}).(BoolOutput))
	})
	t.Run("Float64", func(t *testing.T) {
		t.Parallel()
		assertApplied(t, initialOutput.ApplyT(func(v interface{}) (Float64, error) {
			return Float64(0.0), nil
		}).(Float64Output))
	})
	t.Run("Int", func(t *testing.T) {
		t.Parallel()
		assertApplied(t, initialOutput.ApplyT(func(v interface{}) (Int, error) {
			return Int(0), nil
		}).(IntOutput))
	})
	t.Run("String", func(t *testing.T) {
		t.Parallel()
		assertApplied(t, initialOutput.ApplyT(func(v interface{}) (String, error) {
			return String(""), nil
		}).(StringOutput))
	})
	t.Run("BoolInput", func(t *testing.T) {
		t.Parallel()
		assertApplied(t, initialOutput.ApplyT(func(v interface{}) (BoolInput, error) {
			return Bool(false), nil
		}).(BoolOutput))
	})
	t.Run("Float64Input", func(t *testing.T) {
		t.Parallel()
		assertApplied(t, initialOutput.ApplyT(func(v interface{}) (Float64Input, error) {
			return Float64(0.0), nil
		}).(Float64Output))
	})
	t.Run("IntInput", func(t *testing.T) {
		t.Parallel()
		assertApplied(t, initialOutput.ApplyT(func(v interface{}) (IntInput, error) {
			return Int(0), nil
		}).(IntOutput))
	})
	t.Run("StringInput", func(t *testing.T) {
		t.Parallel()
		assertApplied(t, initialOutput.ApplyT(func(v interface{}) (StringInput, error) {
			return String(""), nil
		}).(StringOutput))
	})
}

func TestResolveOutputToOutput(t *testing.T) {
	t.Parallel()

	// Test that resolving an output to an output yields the value, not the output.
	{
		out, resolve, _ := NewOutput()
		go func() {
			other, resolveOther, _ := NewOutput()
			resolve(other)
			go func() { resolveOther(99) }()
		}()
		assertApplied(t, out.ApplyT(func(v interface{}) (interface{}, error) {
			assert.Equal(t, v, 99)
			return nil, nil
		}))
	}
	// Similarly, test that resolving an output to a rejected output yields an error.
	{
		out, resolve, _ := NewOutput()
		go func() {
			other, _, rejectOther := NewOutput()
			resolve(other)
			go func() { rejectOther(errors.New("boom")) }()
		}()
		v, _, _, _, err := await(out)
		assert.NotNil(t, err)
		assert.Nil(t, v)
	}
}

// Test that ToOutput works with a struct type.
func TestToOutputStruct(t *testing.T) {
	t.Parallel()

	out := ToOutput(nestedTypeInputs{Foo: String("bar"), Bar: Int(42)})
	_, ok := out.(nestedTypeOutput)
	assert.True(t, ok)

	v, known, secret, deps, err := await(out)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)
	assert.Equal(t, nestedType{Foo: "bar", Bar: 42}, v)

	out = ToOutput(out)
	_, ok = out.(nestedTypeOutput)
	assert.True(t, ok)

	v, known, secret, deps, err = await(out)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)

	assert.NoError(t, err)
	assert.Equal(t, nestedType{Foo: "bar", Bar: 42}, v)

	out = ToOutput(nestedTypeInputs{Foo: ToOutput(String("bar")).(StringInput), Bar: ToOutput(Int(42)).(IntInput)})
	_, ok = out.(nestedTypeOutput)
	assert.True(t, ok)

	v, known, secret, deps, err = await(out)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)
	assert.Equal(t, nestedType{Foo: "bar", Bar: 42}, v)
}

type arrayLenInput Array

func (arrayLenInput) ElementType() reflect.Type {
	return Array{}.ElementType()
}

func (i arrayLenInput) ToIntOutput() IntOutput {
	return i.ToIntOutputWithContext(context.Background())
}

func (i arrayLenInput) ToIntOutputWithContext(ctx context.Context) IntOutput {
	return ToOutput(i).ApplyT(func(arr []interface{}) int {
		return len(arr)
	}).(IntOutput)
}

func (i arrayLenInput) ToIntPtrOutput() IntPtrOutput {
	return i.ToIntPtrOutputWithContext(context.Background())
}

func (i arrayLenInput) ToIntPtrOutputWithContext(ctx context.Context) IntPtrOutput {
	return ToOutput(i).ApplyT(func(arr []interface{}) *int {
		v := len(arr)
		return &v
	}).(IntPtrOutput)
}

// Test that ToOutput converts inputs appropriately.
func TestToOutputConvert(t *testing.T) {
	t.Parallel()

	out := ToOutput(nestedTypeInputs{Foo: ID("bar"), Bar: arrayLenInput{Int(42)}})
	_, ok := out.(nestedTypeOutput)
	assert.True(t, ok)

	v, known, secret, deps, err := await(out)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)
	assert.Equal(t, nestedType{Foo: "bar", Bar: 1}, v)
}

// Test that ToOutput correctly handles nested inputs and outputs when the argument is an input or interface{}.
func TestToOutputAny(t *testing.T) {
	t.Parallel()

	type args struct {
		S StringInput
		I IntInput
		A Input
	}

	out := ToOutput(&args{
		S: ID("hello"),
		I: Int(42).ToIntOutput(),
		A: Map{"world": Bool(true).ToBoolOutput()},
	})
	_, ok := out.(AnyOutput)
	assert.True(t, ok)

	v, known, secret, deps, err := await(out)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)

	argsV := v.(*args)

	si, ok := argsV.S.(ID)
	assert.True(t, ok)
	assert.Equal(t, ID("hello"), si)

	io, ok := argsV.I.(IntOutput)
	assert.True(t, ok)
	assert.Equal(t, uint32(outputResolved), io.state)
	assert.Equal(t, 42, io.value)

	ai, ok := argsV.A.(Map)
	assert.True(t, ok)

	bo, ok := ai["world"].(BoolOutput)
	assert.True(t, ok)
	assert.Equal(t, uint32(outputResolved), bo.getState().state)
	assert.Equal(t, true, bo.value)
}

func TestToOutputAnyDeps(t *testing.T) {
	t.Parallel()

	type args struct {
		S StringInput
		I IntInput
		A Input
		R Resource
	}

	stringDep1, stringDep2 := &ResourceState{}, &ResourceState{}
	stringOut := StringOutput{newOutputState(nil, reflect.TypeOf(""), stringDep1)}
	go func() {
		stringOut.resolve("a stringy output", true, false, []Resource{stringDep2})
	}()

	intDep1, intDep2 := &ResourceState{}, &ResourceState{}
	intOut := IntOutput{newOutputState(nil, reflect.TypeOf(0), intDep1)}
	go func() {
		intOut.resolve(42, true, false, []Resource{intDep2})
	}()

	boolDep1, boolDep2 := &ResourceState{}, &ResourceState{}
	boolOut := BoolOutput{newOutputState(nil, reflect.TypeOf(true), boolDep1)}
	go func() {
		boolOut.resolve(true, true, false, []Resource{boolDep2})
	}()

	res := &ResourceState{}
	urnOut := URNOutput{newOutputState(nil, reflect.TypeOf(URN("")), res)}
	go func() {
		urnOut.resolve(URN("foo"), true, false, nil)
	}()
	res.urn = urnOut

	out := ToOutput(&args{
		S: stringOut,
		I: intOut,
		A: Map{"world": boolOut},
		R: res,
	})
	_, ok := out.(AnyOutput)
	assert.True(t, ok)

	v, known, secret, deps, err := await(out)
	assert.True(t, known)
	assert.False(t, secret)
	assert.ElementsMatch(t, []Resource{stringDep1, stringDep2, intDep1, intDep2, boolDep1, boolDep2}, deps)
	assert.NoError(t, err)

	argsV := v.(*args)

	so, ok := argsV.S.(StringOutput)
	assert.True(t, ok)
	assert.Equal(t, uint32(outputResolved), so.state)
	assert.Equal(t, "a stringy output", so.value)
	assert.ElementsMatch(t, []Resource{stringDep1, stringDep2}, so.deps)

	io, ok := argsV.I.(IntOutput)
	assert.True(t, ok)
	assert.Equal(t, uint32(outputResolved), io.state)
	assert.Equal(t, 42, io.value)
	assert.ElementsMatch(t, []Resource{intDep1, intDep2}, io.deps)

	ai, ok := argsV.A.(Map)
	assert.True(t, ok)

	bo, ok := ai["world"].(BoolOutput)
	assert.True(t, ok)
	assert.Equal(t, uint32(outputResolved), bo.getState().state)
	assert.Equal(t, true, bo.value)
	assert.ElementsMatch(t, []Resource{boolDep1, boolDep2}, bo.deps)

	ro := argsV.R
	urn, known, secret, deps, err := await(ro.URN())
	assert.Equal(t, URN("foo"), urn)
	assert.True(t, known)
	assert.False(t, secret)
	assert.ElementsMatch(t, []Resource{res}, deps)
	assert.NoError(t, err)
}

type args struct {
	S string
	I int
	A interface{}
}

type argsInputs struct {
	S StringInput
	I IntInput
	A Input
}

func (*argsInputs) ElementType() reflect.Type {
	return reflect.TypeOf((*args)(nil))
}

// Test that ToOutput correctly handles nested inputs when the argument is an input with no corresponding output type.
func TestToOutputInputAny(t *testing.T) {
	t.Parallel()

	out := ToOutput(&argsInputs{
		S: ID("hello"),
		I: Int(42),
		A: Map{"world": Bool(true).ToBoolOutput()},
	})
	_, ok := out.(AnyOutput)
	assert.True(t, ok)

	v, known, secret, deps, err := await(out)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)

	assert.Equal(t, &args{
		S: "hello",
		I: 42,
		A: map[string]interface{}{"world": true},
	}, v)
}

// Test that Unsecret will return an Output that has an unwrapped secret
func TestUnsecret(t *testing.T) {
	t.Parallel()

	s := ToSecret(String("foo"))
	// assert that secret is immediately secret
	assert.True(t, IsSecret(s))

	unS := Unsecret(s)
	// assert that we do not have a secret
	assert.False(t, IsSecret(unS))

	errChan := make(chan error)
	resultChan := make(chan string)
	secretChan := make(chan bool)

	unS.ApplyT(func(v interface{}) (string, error) {
		// assert secretness after the output resolves
		secretChan <- IsSecret(unS)
		val := v.(string)
		if val == "foo" {
			// validate the value
			resultChan <- val
		} else {
			errChan <- fmt.Errorf("invalid result: %v", val)
		}
		return val, nil
	})

	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			assert.Nil(t, err)
			break
		case r := <-resultChan:
			assert.Equal(t, "foo", r)
			break
		case isSecret := <-secretChan:
			assert.False(t, isSecret)
			break
		}
	}
}

// Test that SecretT sets appropriate internal state and that IsSecret appropriately reads it.
func TestSecrets(t *testing.T) {
	t.Parallel()

	s := ToSecret(String("foo"))
	// assert that secret is immediately secret
	assert.True(t, IsSecret(s))

	errChan := make(chan error)
	resultChan := make(chan string)
	secretChan := make(chan bool)

	s.ApplyT(func(v interface{}) (string, error) {
		// assert secretness after the output resolves
		secretChan <- IsSecret(s)
		val := v.(string)
		if val == "foo" {
			// validate the value
			resultChan <- val
		} else {
			errChan <- fmt.Errorf("invalid result: %v", val)
		}
		return val, nil
	})

	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			assert.Nil(t, err)
			break
		case r := <-resultChan:
			assert.Equal(t, "foo", r)
			break
		case isSecret := <-secretChan:
			assert.True(t, isSecret)
			break
		}
	}

}

// Test that secretness is properly bubbled up with all/apply.
func TestSecretApply(t *testing.T) {
	t.Parallel()

	s1 := ToSecret(String("foo"))
	// assert that secret is immediately secret
	assert.True(t, IsSecret(s1))
	s2 := StringInput(String("bar"))

	errChan := make(chan error)
	resultChan := make(chan string)
	secretChan := make(chan bool)

	s := All(s1, s2).ApplyT(func(v interface{}) (string, error) {
		val := v.([]interface{})
		return val[0].(string) + val[1].(string), nil
	})
	s.ApplyT(func(v interface{}) (string, error) {
		// assert secretness after the output resolves
		secretChan <- IsSecret(s)
		val := v.(string)
		if val == "foobar" {
			// validate the value
			resultChan <- val
		} else {
			errChan <- fmt.Errorf("invalid result: %v", val)
		}
		return val, nil
	})

	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			assert.Nil(t, err)
			break
		case r := <-resultChan:
			assert.Equal(t, "foobar", r)
			break
		case isSecret := <-secretChan:
			assert.True(t, isSecret)
			break
		}
	}
}

// Test that secretness is properly bubbled up with all/apply that delays its execution.
func TestSecretApplyDelayed(t *testing.T) {
	t.Parallel()

	// We run multiple tests here to increase the likelihood of a hypothetical race
	// condition triggering. As with all concurrency tests, its not a 100% guarantee.
	for i := 0; i < 10 && !t.Failed(); i++ {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			s1 := String("foo").ToStringOutput().ApplyT(func(s string) StringOutput {
				time.Sleep(time.Millisecond * 5)
				return ToSecret(String("bar")).(StringOutput)
			})
			// assert that s1 is secret.
			assert.True(t, IsSecret(s1))
		})
	}
}

func TestNil(t *testing.T) {
	t.Parallel()

	ao := Any(nil)
	v, known, secret, deps, err := await(ao)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)
	assert.Equal(t, nil, v)

	o := ToOutput(nil)
	v, known, secret, deps, err = await(o)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)
	assert.Equal(t, nil, v)

	o = ToOutput(ao)
	v, known, secret, deps, err = await(o)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)
	assert.Equal(t, nil, v)

	ao = ToOutput("").ApplyT(func(v string) interface{} {
		return nil
	}).(AnyOutput)
	v, known, secret, deps, err = await(ao)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)
	assert.Equal(t, nil, v)

	bo := ao.ApplyT(func(x interface{}) bool {
		return x == nil
	})
	v, known, secret, deps, err = await(bo)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NoError(t, err)
	assert.Equal(t, true, v)
}

// Test that dependencies flow through all/apply.
func TestDeps(t *testing.T) {
	t.Parallel()

	stringDep1, stringDep2 := &ResourceState{}, &ResourceState{}
	stringOut := StringOutput{newOutputState(nil, reflect.TypeOf(""), stringDep1)}
	assert.ElementsMatch(t, []Resource{stringDep1}, stringOut.deps)
	go func() {
		stringOut.resolve("hello", true, false, []Resource{stringDep2})
	}()

	boolDep1, boolDep2 := &ResourceState{}, &ResourceState{}
	boolOut := BoolOutput{newOutputState(nil, reflect.TypeOf(true), boolDep1)}
	assert.ElementsMatch(t, []Resource{boolDep1}, boolOut.deps)
	go func() {
		boolOut.resolve(true, true, false, []Resource{boolDep2})
	}()

	a := All(stringOut, boolOut).ApplyT(func(args []interface{}) (string, error) {
		s := args[0].(string)
		b := args[1].(bool)
		return fmt.Sprintf("%s: %v", s, b), nil
	})

	v, known, secret, deps, err := await(a)
	assert.Equal(t, "hello: true", v)
	assert.True(t, known)
	assert.False(t, secret)
	assert.ElementsMatch(t, []Resource{stringDep1, stringDep2, boolDep1, boolDep2}, deps)
	assert.NoError(t, err)
}

func testMixedWaitGroups(t *testing.T, combine func(o1, o2 Output) Output) {
	var wg1, wg2 workGroup

	o1 := newOutput(&wg1, anyOutputType)
	o2 := newOutput(&wg2, anyOutputType)

	gate := make(chan chan bool)
	combine(o1, o2).ApplyT(func(_ interface{}) interface{} {
		<-gate <- true
		return 0
	})

	wg1Done, wg2Done := false, false
	go func() {
		wg1.Wait()
		wg1Done = true
		wg2.Wait()
		wg2Done = true
	}()

	o1.getState().resolve(0, true, true, nil)
	o2.getState().resolve(0, true, true, nil)

	c := make(chan bool)
	gate <- c
	assert.False(t, wg1Done)
	assert.False(t, wg2Done)
	<-c
	wg1.Wait()
	wg2.Wait()

}

func TestMixedWaitGroupsAll(t *testing.T) {
	t.Parallel()

	testMixedWaitGroups(t, func(o1, o2 Output) Output {
		return All(o1, o2)
	})
}

func TestMixedWaitGroupsAny(t *testing.T) {
	t.Parallel()

	testMixedWaitGroups(t, func(o1, o2 Output) Output {
		return Any(struct{ O1, O2 Output }{o1, o2})
	})
}

func TestMixedWaitGroupsApply(t *testing.T) {
	t.Parallel()

	testMixedWaitGroups(t, func(o1, o2 Output) Output {
		return o1.ApplyT(func(_ interface{}) interface{} {
			return o2
		})
	})
}

type Foo interface {
}

type FooInput interface {
	Input

	ToFooOutput() Output
}
type FooArgs struct {
}

func (FooArgs) ElementType() reflect.Type {
	return nil
}

func TestRegisterInputType(t *testing.T) {
	t.Parallel()

	assert.PanicsWithError(t, "expected string to be an interface", func() {
		RegisterInputType(reflect.TypeOf(""), FooArgs{})
	})
	assert.PanicsWithError(t, "expected pulumi.Foo to implement pulumi.Input", func() {
		RegisterInputType(reflect.TypeOf((*Foo)(nil)).Elem(), FooArgs{})
	})
	assert.PanicsWithError(t, "expected pulumi.FooArgs to implement interface pulumi.FooInput", func() {
		RegisterInputType(reflect.TypeOf((*FooInput)(nil)).Elem(), FooArgs{})
	})
}

func TestAll(t *testing.T) {
	t.Parallel()

	aStringInput := String("Test")
	aStringPtrInput := StringPtr("Hello World")
	aStringOutput := String("Frob").ToStringOutput()

	a := All(aStringInput).ApplyT(func(args []interface{}) (string, error) {
		a := args[0].(string)
		return a, nil
	}).(StringOutput)

	v, known, secret, deps, err := await(a)
	assert.Equal(t, "Test", v)
	assert.True(t, known)
	assert.False(t, secret)
	assert.ElementsMatch(t, []Resource{}, deps)
	assert.NoError(t, err)

	a = All(aStringPtrInput).ApplyT(func(args []interface{}) (string, error) {
		a := args[0].(*string)
		return *a, nil
	}).(StringOutput)

	v, known, secret, deps, err = await(a)
	assert.Equal(t, "Hello World", v)
	assert.True(t, known)
	assert.False(t, secret)
	assert.ElementsMatch(t, []Resource{}, deps)
	assert.NoError(t, err)

	a = All(aStringOutput).ApplyT(func(args []interface{}) (string, error) {
		a := args[0].(string)
		return a, nil
	}).(StringOutput)

	v, known, secret, deps, err = await(a)
	assert.Equal(t, "Frob", v)
	assert.True(t, known)
	assert.False(t, secret)
	assert.ElementsMatch(t, []Resource{}, deps)
	assert.NoError(t, err)

	a = All(aStringInput, aStringPtrInput, aStringOutput).ApplyT(func(args []interface{}) (string, error) {
		a := args[0].(string)
		b := args[1].(*string)
		c := args[2].(string)
		return fmt.Sprintf("%s: %s: %s", a, *b, c), nil
	}).(StringOutput)

	v, known, secret, deps, err = await(a)
	assert.Equal(t, "Test: Hello World: Frob", v)
	assert.True(t, known)
	assert.False(t, secret)
	assert.ElementsMatch(t, []Resource{}, deps)
	assert.NoError(t, err)
}

func TestApplyTOutput(t *testing.T) {
	t.Parallel()

	ctx, err := NewContext(context.Background(), RunInfo{})
	assert.Nil(t, err)
	r1 := newSimpleCustomResource(ctx, URN("urn1"), ID("id1"))
	r2 := newSimpleCustomResource(ctx, URN("urn2"), ID("id2"))
	r3 := newSimpleCustomResource(ctx, URN("urn3"), ID("id3"))
	r4 := newSimpleCustomResource(ctx, URN("urn4"), ID("id4"))
	out1 := StringOutput{newOutputState(nil, reflect.TypeOf(""), r1)}
	out2 := IntOutput{newOutputState(nil, reflect.TypeOf(0), r2)}
	go func() {
		out1.resolve("r1 output", true, false, []Resource{r3})
		out2.resolve(42, true, false, []Resource{r4})
	}()
	{
		out3 := out1.ApplyT(func(v string) (IntOutput, error) {
			return out2, nil
		})
		v, _, _, deps, err := out3.getState().await(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 42, v)
		assert.Equal(t, fmt.Sprintf("%v", reflect.TypeOf(v)), "int")
		assert.Len(t, deps, 4)
	}
}

func assertResult(t *testing.T, o Output, expectedValue interface{}, expectedKnown, expectedSecret bool, expectedDeps ...CustomResource) {
	t.Helper()
	v, known, secret, deps, err := o.getState().await(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expectedValue, v, "values do not match")
	assert.Equal(t, expectedKnown, known, "known-ness does not match")
	assert.Equal(t, expectedSecret, secret, "secret-ness does not match")
	depUrns := make([]URN, 0, len(deps))
	for _, v := range deps {
		depUrns = append(depUrns, v.URN().value.(URN))
	}
	var expectedUrns = make([]URN, 0, len(expectedDeps))
	for _, v := range expectedDeps {
		expectedUrns = append(expectedUrns, v.URN().value.(URN))
	}
	assert.ElementsMatch(t, depUrns, expectedUrns)
}

// Test that nested Apply operations accumulate state correctly.
func TestApplyTOutputJoinDeps(t *testing.T) {
	t.Parallel()

	ctx, err := NewContext(context.Background(), RunInfo{})
	assert.Nil(t, err)
	rA := newSimpleCustomResource(ctx, URN("urnA"), ID("idA"))
	rB := newSimpleCustomResource(ctx, URN("urnB"), ID("idB"))

	outA := IntOutput{newOutputState(nil, reflect.TypeOf(0), rA)}
	outB := IntOutput{newOutputState(nil, reflect.TypeOf(0), rB)}

	applyF := func(outA, outB IntOutput) IntOutput {
		return outA.ApplyT(func(v int) (IntOutput, error) {
			return outB, nil
		}).(IntOutput)
	}

	outAB := applyF(outA, outB)

	outA.resolve(3, true, false, []Resource{rA})
	outB.resolve(5, true, false, []Resource{rB})

	assertResult(t, outA, 3, true, false, rA)
	assertResult(t, outAB, 5, true, false, rA, rB)
	assertResult(t, outB, 5, true, false, rB)

}

// Test that nested Apply operations accumulate state correctly.
func TestApplyTOutputJoin(t *testing.T) {
	t.Parallel()

	ctx, err := NewContext(context.Background(), RunInfo{})
	assert.Nil(t, err)
	r1 := newSimpleCustomResource(ctx, URN("urn1"), ID("id1"))
	r2 := newSimpleCustomResource(ctx, URN("urn2"), ID("id2"))
	r3 := newSimpleCustomResource(ctx, URN("urn3"), ID("id3"))

	out1 := IntOutput{newOutputState(nil, reflect.TypeOf(0), r1)}
	out2 := IntOutput{newOutputState(nil, reflect.TypeOf(0), r2)}
	out3 := IntOutput{newOutputState(nil, reflect.TypeOf(0), r3)}

	go func() {
		out1.resolve(2, true, false, []Resource{r1})
		out2.resolve(3, false, false, []Resource{r2}) // value set but known => output.value == nil
		out3.resolve(5, true, true, []Resource{r3})
	}()

	applyF := func(outA, outB IntOutput) IntOutput {
		return outA.ApplyT(func(v int) (IntOutput, error) {
			return outB, nil
		}).(IntOutput)
	}

	out12 := applyF(out1, out2)
	out123 := applyF(out12, out3)

	out23 := applyF(out2, out3)
	out231 := applyF(out23, out1)

	out31 := applyF(out3, out1)
	out312 := applyF(out31, out2)

	assertResult(t, out1, 2, true, false, r1)
	assertResult(t, out12, nil, false, false, r1, r2)
	assertResult(t, out123, nil, false, false, r1, r2) /* out2 is unknown, hiding out3 */

	/* out2 is unknown, early exit hides all nested outputs */
	assertResult(t, out2, nil, false, false, r2)
	assertResult(t, out23, nil, false, false, r2)
	assertResult(t, out231, nil, false, false, r2)

	assertResult(t, out3, 5, true, true, r3)
	assertResult(t, out31, 2, true, true, r3, r1)
	assertResult(t, out312, nil, false, true, r3, r1, r2) /* out2 is unknown, hiding the output */
}

func TestTypeCoersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    interface{}
		expected interface{}
		err      string
	}{
		{"foo", "foo", ""},
		{"foo", 0, "expected value of type int, not string"},
		{
			map[string]interface{}{
				"foo":  "bar",
				"fizz": "buzz",
			},
			map[string]string{
				"foo":  "bar",
				"fizz": "buzz",
			},
			"",
		},
		{
			map[string]interface{}{
				"foo":  "bar",
				"fizz": 8,
			},
			map[string]string{
				"foo":  "bar",
				"fizz": "buzz",
			},
			`["fizz"]: expected value of type string, not int`,
		},
		{
			[]interface{}{1, 2, 3},
			[]int{1, 2, 3},
			"",
		},
		{
			[]interface{}{1, "two", 3},
			[]int{1, 2, 3},
			`[1]: expected value of type int, not string`,
		},
		{
			[]interface{}{
				map[string]interface{}{
					"fizz":     []interface{}{3, 15},
					"buzz":     []interface{}{5, 15},
					"fizzbuzz": []interface{}{15},
				},
				map[string]interface{}{},
			},
			[]map[string][]int{
				{
					"fizz":     {3, 15},
					"buzz":     {5, 15},
					"fizzbuzz": {15},
				},
				{},
			},
			"",
		},
		{
			[]interface{}{
				map[string]interface{}{
					"fizz":     []interface{}{3, 15},
					"buzz":     []interface{}{"5", 15},
					"fizzbuzz": []interface{}{15},
				},
				map[string]interface{}{},
			},
			[]map[string][]int{
				{
					"fizz":     {3, 15},
					"buzz":     {5, 15},
					"fizzbuzz": {15},
				},
				{},
			},
			`[0]: ["buzz"]: [0]: expected value of type int, not string`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%v->%v", tt.input, tt.expected), func(t *testing.T) {
			t.Parallel()
			dstT := reflect.TypeOf(tt.expected)
			val, err := coerceTypeConversion(tt.input, dstT)
			if tt.err == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
			} else {
				assert.EqualError(t, err, tt.err)
			}
		})
	}
}

func TestJSONMarshalBasic(t *testing.T) {
	t.Parallel()

	out, resolve, _ := NewOutput()
	go func() {
		resolve([]int{0, 1})
	}()
	json := JSONMarshal(out)
	v, known, secret, deps, err := await(json)
	assert.Nil(t, err)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NotNil(t, v)
	assert.Equal(t, "[0,1]", v.(string))
}

func TestJSONMarshalNested(t *testing.T) {
	t.Parallel()

	a, resolvea, _ := NewOutput()
	go func() {
		resolvea(0)
	}()
	b, resolveb, _ := NewOutput()
	go func() {
		resolveb(1)
	}()
	out, resolve, _ := NewOutput()
	go func() {
		resolve([]Output{a, b})
	}()
	json := JSONMarshal(out)
	v, known, secret, deps, err := await(json)
	assert.Equal(t, "json: error calling MarshalJSON for type pulumi.AnyOutput: Outputs can not be marshaled to JSON", err.Error())
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.Nil(t, v)
}

func TestJSONUnmarshalBasic(t *testing.T) {
	t.Parallel()

	out, resolve, _ := NewOutput()
	go func() {
		resolve("[0, 1]")
	}()
	str := out.ApplyT(func(str interface{}) (string, error) {
		return str.(string), nil
	}).(StringOutput)
	json := JSONUnmarshal(str)
	v, known, secret, deps, err := await(json)
	assert.Nil(t, err)
	assert.True(t, known)
	assert.False(t, secret)
	assert.Nil(t, deps)
	assert.NotNil(t, v)
	assert.Equal(t, []interface{}{0.0, 1.0}, v.([]interface{}))
}

func TestApplyTSignatureMismatch(t *testing.T) {
	t.Parallel()

	var pval interface{}
	func() {
		defer func() { pval = recover() }()

		Int(42).ToIntOutput().ApplyT(func(string) string {
			t.Errorf("This function should not be called")
			return ""
		})
	}()
	require.NotNil(t, pval, "function did not panic")

	msg := fmt.Sprint(pval)
	assert.Regexp(t, `applier defined at .+?types_test\.go:\d+`, msg)
}

func TestApplier_Call(t *testing.T) {
	t.Parallel()

	stringType := reflect.TypeOf("")

	t.Run("minimal", func(t *testing.T) {
		t.Parallel()

		ap, err := newApplier(func(s string) int {
			assert.Equal(t, "hello", s)
			return 42
		}, stringType)
		require.NoError(t, err)

		o, err := ap.Call(context.Background(), reflect.ValueOf("hello"))
		require.NoError(t, err)
		assert.Equal(t, int64(42), o.Int())
	})

	t.Run("context", func(t *testing.T) {
		t.Parallel()

		giveCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ap, err := newApplier(func(ctx context.Context, s string) int {
			// == check because we want an exact reference match,
			// not a deep equals.
			assert.True(t, ctx == giveCtx, "context must match")
			assert.Equal(t, "hello", s)
			return 42
		}, stringType)
		require.NoError(t, err)

		o, err := ap.Call(giveCtx, reflect.ValueOf("hello"))
		require.NoError(t, err)
		assert.Equal(t, int64(42), o.Int())
	})

	t.Run("error/success", func(t *testing.T) {
		t.Parallel()

		ap, err := newApplier(func(string) (int, error) {
			return 42, nil
		}, stringType)
		require.NoError(t, err)

		o, err := ap.Call(context.Background(), reflect.ValueOf("hello"))
		require.NoError(t, err)
		assert.Equal(t, int64(42), o.Int())
	})

	t.Run("error/failure", func(t *testing.T) {
		t.Parallel()

		giveErr := errors.New("great sadness")

		ap, err := newApplier(func(string) (int, error) {
			return 0, giveErr
		}, stringType)
		require.NoError(t, err)

		_, err = ap.Call(context.Background(), reflect.ValueOf("hello"))
		// == check because we want an exact reference match,
		// not a deep equals.
		assert.True(t, err == giveErr, "error must match")
	})
}

func TestNewApplier_errors(t *testing.T) {
	t.Parallel()

	stringType := reflect.TypeOf("")
	tests := []struct {
		desc string
		give interface{}

		// Part of the error message expected in return.
		wantErr string
	}{
		{
			desc:    "no params",
			give:    func() int { return 0 },
			wantErr: "applier must accept exactly one or two parameters, got 0",
		},
		{
			desc:    "single param bad input",
			give:    func(int) int { return 0 },
			wantErr: "applier's first input parameter must be assignable from string, got int",
		},
		{
			desc:    "two params bad context",
			give:    func(int, string) int { return 0 },
			wantErr: "applier's first input parameter must be assignable from context.Context, got int",
		},
		{
			desc:    "two params bad input",
			give:    func(context.Context, int) int { return 0 },
			wantErr: "applier's second input parameter must be assignable from string, got int",
		},
		{
			desc:    "three params",
			give:    func(context.Context, string, int) int { return 0 },
			wantErr: "applier must accept exactly one or two parameters, got 3",
		},
		{
			desc:    "no returns",
			give:    func(string) {},
			wantErr: "applier must return exactly one or two values, got 0",
		},
		{
			desc:    "two returns bad error",
			give:    func(string) (int, int) { return 0, 0 },
			wantErr: "applier's second return type must be assignable to error, got int",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			_, err := newApplier(tt.give, stringType)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestApplyTCoerce(t *testing.T) {
	t.Parallel()

	t.Run("ID-string", func(t *testing.T) {
		t.Parallel()

		o := ID("hello").ToIDOutput()
		assertApplied(t, o.ApplyT(func(s string) (interface{}, error) {
			assert.Equal(t, "hello", s)
			return nil, nil
		}))
	})

	t.Run("string-ID", func(t *testing.T) {
		t.Parallel()

		o := String("world").ToStringOutput()
		assertApplied(t, o.ApplyT(func(id ID) (interface{}, error) {
			assert.Equal(t, "world", string(id))
			return nil, nil
		}))
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()

		type Foo struct{ v int }
		type Bar Foo

		type FooOutput struct{ *OutputState }

		o := FooOutput{newOutputState(nil, reflect.TypeOf(Foo{}))}
		go o.resolve(Foo{v: 42}, true, false, nil)

		assertApplied(t, o.ApplyT(func(b Bar) (interface{}, error) {
			assert.Equal(t, 42, b.v)
			return nil, nil
		}))
	})
}

// Verifies that ApplyT does not allow applierse where the conversion
// would change the undedrlying representation.
func TestApplyTCoerceRejectDifferentKinds(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		String("foo").ToStringOutput().ApplyT(func([]byte) int {
			t.Error("Should not be called")
			return 42
		})
	}, "string-[]byte should not be allowed")

	assert.Panics(t, func() {
		Int(42).ToIntOutput().ApplyT(func(string) int {
			t.Error("Should not be called")
			return 42
		})
	}, "int-string should not be allowed")
}
