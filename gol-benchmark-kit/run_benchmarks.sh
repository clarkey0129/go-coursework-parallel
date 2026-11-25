#!/usr/bin/env bash

set -euo pipefail

#name and distributor file for each strategy
STRATEGIES=(
  "double-buffer gol/distributor.go"
  "heap heap-allocation/distributor.go"
  "map-reduce map-reduce/distributor.go"
)
COUNT=${COUNT:-10} #runs per configuration

KIT=$(dirname "$0")
OUT=bench-results
mkdir -p "$OUT"
cp "$KIT/benchmark_test.go" . #the benchmark has to sit in the project root next to go.mod

#keeps a copy of double buffer distributor and puts it back at the end, even if the script is stopped
cp gol/distributor.go "$OUT/distributor.go.backup"
trap 'cp "$OUT/distributor.go.backup" gol/distributor.go; rm -f benchmark_test.go; echo "restored gol/distributor.go"' EXIT

#records the machine for the report
{ date; go version; lscpu | grep -E "Model name|^CPU\(s\)|Thread|Core|Socket|L2|L3"; free -g | head -2; } > "$OUT/machine.txt" 2>/dev/null || true

for s in "${STRATEGIES[@]}"; do
  read -r name file <<< "$s"
  echo ">>> $name ($file)"
  [ "$file" = gol/distributor.go ] || cp "$file" gol/distributor.go #swaps the strategy in (the main one is already there)

  #correctness first
  go test ./tests -v -race > "$OUT/tests_$name.txt" 2>&1 || { echo "tests failed for $name, see $OUT/tests_$name.txt"; exit 1; }

  #benchmark with -count 10 and -benchmem for allocations
  go test -run '^$' -bench . -benchtime 1x -count "$COUNT" -benchmem -timeout 0 | tee "$OUT/$name.out"

  #benchstat summary (plot.py reads these, needs internet the first time)
  go run golang.org/x/perf/cmd/benchstat@latest -format csv "$OUT/$name.out" | tee "$OUT/$name.csv"

  cp "$OUT/distributor.go.backup" gol/distributor.go #puts the main distributor back before the next one
done

#execution trace with the skeleton's TestTrace (64x64, 4 workers, 10 turns)
go test ./tests -run TestTrace > /dev/null 2>&1 || true
mv trace.out "$OUT/trace_TestTrace.out" 2>/dev/null || true #the tests change into the project root so trace.out ends up there

#trace of the full benchmark at 512x512 with 4 workers using go test -trace
go test -run '^$' -bench 'BenchmarkGol/512x512x1000-4$' -benchtime 1x -trace "$OUT/trace_512_4workers.out" > /dev/null 2>&1 || true

python3 "$KIT/plot.py" "$OUT" #bar chart, and writes explorer.csv
