gol-benchmark-kit

what it is

  benchmark_test.go   the BenchmarkGol from the report guidance page
                      also runs 64x64, 128x128 and 256x256
                      and reports gc cycles and gc pause time per run, for the memory strategy comparison
  run_benchmarks.sh   for each strategy: go test ./tests -v -race (benchmarking page, section 4)
                      then go test -run '^$' -bench . -benchtime 1x -count 10 -benchmem | tee
                      then benchstat -format csv | tee
                      then TestTrace, plus one go test -trace at 512x512 with 4 workers
  plot.py             reads the time table from each benchstat csv,
                      draws the bar chart of time against worker threads at 512x512 for all three strategies,
                      and writes explorer.csv for the results explorer on the benchmarking page

to run: ./gol-benchmark-kit/run_benchmarks.sh

what you get in bench-results
  machine.txt            cpu, cores, go version
  tests_<strategy>.txt   race test output for each strategy
  <strategy>.out         raw go test output
  <strategy>.csv         benchstat summary: median time, memory and gc, each in its own table
  explorer.csv           every run as engine,size,threads,ns_per_turn, paste it into the results explorer
                         for the time per turn and speedup charts and the efficiency and karp-flatt table
  bar_512.png            bar chart of time against workers at 512x512, all three strategies
  trace_TestTrace.out    the skeleton's trace (64x64, 4 workers, 10 turns): go tool trace bench-results/trace_TestTrace.out
  trace_512_4workers.out the full 1000-turn benchmark run at 512x512 with 4 workers

note:
  the script swaps each strategy into gol/distributor.go and puts double buffer back at the end
