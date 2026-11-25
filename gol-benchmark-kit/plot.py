import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
import re #DIFF: for reading the raw go test output for explorer.csv
import sys #DIFF: so the script can be told where the results are

out = sys.argv[1] if len(sys.argv) > 1 else 'bench-results' #DIFF: folder the run script writes to

#Read in the saved CSV data.
rows = []
for strategy in ['double-buffer', 'heap', 'map-reduce']: #one benchstat csv per memory strategy
    try:
        lines = open(f'{out}/{strategy}.csv').read().splitlines()
    except FileNotFoundError:
        continue
    in_time_table = False
    for line in lines:
        if line.startswith(',sec/op'): #start of the time table
            in_time_table = True
        elif line.startswith('geomean'): #end of the time table
            in_time_table = False
        elif in_time_table and 'Gol/' in line:
            name, time, rng = line.split(',')[:3]
            rows.append([strategy, name, float(time) * 1e9, rng]) #back to nanoseconds
benchmark_data = pd.DataFrame(rows, columns=['strategy', 'name', 'time', 'range'])

# Go stores benchmark results in nanoseconds. Convert all results to seconds.
benchmark_data['time'] /= 1e+9

# Use the name of the benchmark to extract the number of worker threads used.
benchmark_data['threads'] = benchmark_data['name'].str.extract(r'Gol/\d+x\d+x\d+-(\d+)-\d+').apply(pd.to_numeric)
benchmark_data['cpu_cores'] = benchmark_data['name'].str.extract(r'Gol/\d+x\d+x\d+-\d+-(\d+)').apply(pd.to_numeric)
benchmark_data['size'] = benchmark_data['name'].str.extract(r'Gol/(\d+)x').apply(pd.to_numeric) #DIFF: image size, like array_length in the merge sort plot.py

print(benchmark_data)

# Plot a bar chart.
ax = sns.barplot(data=benchmark_data[benchmark_data['size'] == 512], x='threads', y='time', hue='strategy') #DIFF: 512x512 only, one bar per strategy

# Set descriptive axis lables.
ax.set(xlabel='Worker threads used', ylabel='Time taken (s)')

#DIFF: writes explorer.csv for the results explorer on the benchmarking page
#it wants engine,size,threads,ns_per_turn with one row per run, so this reads every run from the raw go test output
line = re.compile(r'BenchmarkGol/(\d+)x\d+x(\d+)-(\d+)-\d+\s+\d+\s+(\d+) ns/op')
with open(f'{out}/explorer.csv', 'w') as f:
    f.write('engine,size,threads,ns_per_turn\n')
    for engine in ['double-buffer', 'heap', 'map-reduce']:
        try:
            text = open(f'{out}/{engine}.out').read()
        except FileNotFoundError:
            continue
        text = re.sub(r'\d{4}/\d\d/\d\d \d\d:\d\d:\d\d [^\n]*\n', '', text) #removes the [IO] log lines the program prints
        for m in line.finditer(text):
            size, turns, threads, ns = map(int, m.groups())
            f.write(f'{engine},{size},{threads},{ns // turns}\n') #time for one turn in nanoseconds
print(f'wrote {out}/explorer.csv, paste it into the results explorer on the benchmarking page')

# Display the full figure.
plt.savefig(f'{out}/bar_512.png', dpi=200) #DIFF: also saves it for the report
plt.show()
