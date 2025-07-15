mod benchmark;

use std::time::Instant;

use criterion::Criterion;
use simd_poc::vector::{simd::*, Vector};

fn main() {
    let v1 = Vector::with_length(8, 1.0);
    let v2 = Vector::with_length(8, 2.0);
    let mut simd_result = Vector::with_length(8, 0.0);

    benchmark::bench_vector_operations(&mut Criterion::default());
    let start = Instant::now();
    let scalar_result = v1.clone() + v2.clone();
    println!("Time taken: {:.2?}", start.elapsed());

    // SIMD addition
    println!("\nSIMD Addition:");
    let start = Instant::now();
    add_simd(&v1, &v2, &mut simd_result);
    println!("Time taken: {:.2?}", start.elapsed());

    // Verify results
    assert_eq!(scalar_result.as_slice(), simd_result.as_slice());
    println!("\nResults match between scalar and SIMD implementations!");

    // Run benchmarks
    println!("\nRunning benchmarks...");
    let mut c = Criterion::default();
    c = c.warm_up_time(std::time::Duration::from_secs(1));
    c = c.measurement_time(std::time::Duration::from_secs(3));
    c = c.sample_size(30);

    // Run the benchmark
    benchmark::bench_vector_operations(&mut c);
}
