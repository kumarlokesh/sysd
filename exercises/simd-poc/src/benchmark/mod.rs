use crate::Vector;
use crate::{add_simd, create_simd_vector, dot_simd, mul_simd};
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};

pub mod compare;

pub use compare::bench_scalar_vs_simd;

pub fn bench_vector_operations(c: &mut Criterion) {
    let sizes = [1024, 16384];

    for &size in &sizes {
        let v1 = Vector::random(size, 0.0, 100.0);
        let v2 = Vector::random(size, 0.0, 100.0);
        let mut simd_result = create_simd_vector(size, 0.0);

        let mut group = c.benchmark_group("vector_operations");
        group.warm_up_time(std::time::Duration::from_secs(1));
        group.measurement_time(std::time::Duration::from_secs(3));
        group.sample_size(30);
        group.throughput(Throughput::Elements(size as u64));

        // Vector addition
        group.bench_with_input(BenchmarkId::new("add", size), &size, |b, _| {
            b.iter(|| v1.clone() + v2.clone())
        });

        // SIMD addition
        group.bench_with_input(BenchmarkId::new("simd_add", size), &size, |b, _| {
            b.iter(|| add_simd(&v1, &v2, &mut simd_result))
        });

        // Vector multiplication
        group.bench_with_input(BenchmarkId::new("mul", size), &size, |b, _| {
            b.iter(|| v1.clone() * v2.clone())
        });

        // SIMD multiplication
        group.bench_with_input(BenchmarkId::new("simd_mul", size), &size, |b, _| {
            b.iter(|| mul_simd(&v1, &v2, &mut simd_result))
        });

        // Dot product
        group.bench_with_input(BenchmarkId::new("dot", size), &size, |b, _| {
            b.iter(|| v1.dot(&v2))
        });

        // SIMD dot product
        group.bench_with_input(BenchmarkId::new("simd_dot", size), &size, |b, _| {
            b.iter(|| unsafe { dot_simd(&v1, &v2) })
        });
        group.finish();
    }
}

criterion_group!(
    name = benches;
    config = Criterion::default()
        .warm_up_time(std::time::Duration::from_secs(1))
        .measurement_time(std::time::Duration::from_secs(3))
        .sample_size(10);
    targets = bench_vector_operations, bench_scalar_vs_simd
);

criterion_main!(benches);
