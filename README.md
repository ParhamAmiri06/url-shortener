## phase 3
### BenchmarkShorten-24               846679              1267 ns/op            1856 B/op         23 allocs/op
### BenchmarkRedirect-24             1922692               637.1 ns/op          1265 B/op         13 allocs/op
- Profile captured 3050ms of real time and 3910ms of total CPU sample time
Benchmarking took 70 percent of CPU work while Go runtime (mostly garbage collection)took 30 percent
Test time was split into 1.6s for the redirect benchmark and 1.1s for the shorten benchmark
Almost all redirect benchmark time went to the core http redirect function which took 1510ms
The shorten benchmark bottlenecks were JSON decoding at 310ms and URL normalization at 200ms

