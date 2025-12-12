package builders

import (
	"reflect"
	"testing"
)

func tailOriginal(a []string) []string {
	if len(a) >= 2 {
		return []string(a)[1:]  // 冗余的类型转换
	}
	return []string{}
}

func tailIdiomatic(a []string) []string {
	if len(a) == 0 {
		return []string{}
	}
	return a[1:]
}

type testCase struct {
	name     string
	input    []string
	expected []string
}

// TestTailEquivalence is the main test function to perform equivalence validation
func TestTailEquivalence(t *testing.T) {
	// Define a set of test cases covering all boundary conditions
	tests := []testCase{
		{
			name:     "Case 1: Empty Slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Case 2: Single Element Slice",
			input:    []string{"A"},
			expected: []string{},
		},
		{
			name:     "Case 3: Two Elements Slice",
			input:    []string{"First", "Second"},
			expected: []string{"Second"},
		},
		{
			name:     "Case 4: Multiple Elements Slice",
			input:    []string{"B", "C", "D", "E", "F"},
			expected: []string{"C", "D", "E", "F"},
		},
		{
			name:     "Case 5: Nil Slice",
			input:    nil, // nil slice
			expected: []string{},
		},
	}

	for _, tc := range tests {
		// Use t.Run to ensure each case runs independently and reports clearly
		t.Run(tc.name, func(t *testing.T) {

			// 1. Get the actual results from both functions
			resultOriginal := tailOriginal(tc.input)
			resultIdiomatic := tailIdiomatic(tc.input)

			// 2. [Optional but recommended] Check if both results meet the expectation, to confirm the test itself is correct
			if !reflect.DeepEqual(resultOriginal, tc.expected) {
				t.Errorf("FAIL - Original Function Result Mismatch:\nExpected: %v\nGot: %v", tc.expected, resultOriginal)
			}

			// 3. Core Validation: Check if the outputs of the two functions are completely equivalent
			if !reflect.DeepEqual(resultOriginal, resultIdiomatic) {
				// Use Fatalf to mark the test as failed and stop the case immediately
				t.Fatalf("CRITICAL FAIL - Functions are NOT Equivalent!\nOriginal Got: %v\nIdiomatic Got: %v",
					resultOriginal, resultIdiomatic)
			}

			// Report success, and log the equivalent result
			t.Logf("PASS - Equivalent Result: %v", resultOriginal)
		})
	}
}
