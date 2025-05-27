#!/bin/bash

echo "🔬 Rate Limiter Algorithm Analysis"
echo "=================================="
echo "Testing CORE ALGORITHMIC DIFFERENCES between rate limiters"
echo ""

echo "⏱️ Time Complexity Analysis"
echo "---------------------------"
echo "Counter: O(1) increment | FixedWindow: O(1) + time | SlidingWindow: O(n) cleanup | TokenBucket: O(1) math"
go test -bench=BenchmarkTimeComplexity -benchmem
echo ""

echo "💾 Memory Complexity Analysis"
echo "-----------------------------"
echo "Counter/FixedWindow/TokenBucket: O(1) | SlidingWindow: O(n) - stores every timestamp"
go test -bench=BenchmarkMemoryComplexity -benchmem
echo ""

echo "🔧 Core Data Structure Operations"
echo "---------------------------------"
echo "What each algorithm actually DOES internally"
go test -bench=BenchmarkDataStructureOperations -benchmem
echo ""

echo "🎯 Algorithmic Accuracy"
echo "-----------------------"
echo "How accurately each algorithm enforces rate limits"
go test -bench=BenchmarkAlgorithmicAccuracy -benchmem
echo ""

echo "🔒 Concurrency Model Differences"
echo "--------------------------------"
echo "Different mutex protection patterns and critical sections"
go test -bench=BenchmarkConcurrencyModel -benchmem
echo ""

echo "🗂️ State Management Strategies"
echo "------------------------------"
echo "Counter: no cleanup | Others: different cleanup strategies"
go test -bench=BenchmarkStateManagement -benchmem
echo ""

echo "🧮 Mathematical Operations"
echo "--------------------------"
echo "Computational complexity of core operations"
go test -bench=BenchmarkMathematicalOperations -benchmem
echo ""

echo "⏳ Wait Implementation Differences"
echo "----------------------------------"
echo "SlidingWindow: timestamp-based | TokenBucket: mathematical calculation"
go test -bench=BenchmarkWaitImplementation -benchmem 