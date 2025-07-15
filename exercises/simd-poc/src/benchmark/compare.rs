use criterion::{BenchmarkId, Criterion, Throughput};

use crate::{add_simd, create_simd_vector, dot_simd, mul_simd, Vector};

#[allow(dead_code)]
pub fn bench_scalar_vs_simd(c: &mut Criterion) {
    let mut group = c.benchmark_group("Scalar vs SIMD");

    // Test vector sizes
    let sizes = [1024, 16384, 1048576];

    for &size in &sizes {
        let v1 = Vector::random(size, 0.0, 100.0);
        let v2 = Vector::random(size, 0.0, 100.0);
        let simd_result = create_simd_vector(size, 0.0);

        group.throughput(Throughput::Elements(size as u64));

        group.bench_with_input(
            BenchmarkId::new("scalar_add", size),
            &(&v1, &v2),
            |b, (v1, v2)| b.iter(|| (*v1).clone() + (*v2).clone()),
        );

        {
            let inputs = (v1.clone(), v2.clone(), simd_result.clone());
            group.bench_with_input(
                BenchmarkId::new("simd_add", size),
                &inputs,
                |b, (v1, v2, result)| {
                    b.iter(|| {
                        let mut result = result.clone();
                        add_simd(v1, v2, &mut result);
                        result
                    })
                },
            );
        }

        group.bench_with_input(
            BenchmarkId::new("scalar_mul", size),
            &(&v1, &v2),
            |b, (v1, v2)| b.iter(|| (*v1).clone() * (*v2).clone()),
        );

        {
            let inputs = (v1.clone(), v2.clone(), simd_result.clone());
            group.bench_with_input(
                BenchmarkId::new("simd_mul", size),
                &inputs,
                |b, (v1, v2, result)| {
                    b.iter(|| {
                        let mut result = result.clone();
                        mul_simd(v1, v2, &mut result);
                        result
                    })
                },
            );
        }

        {
            let inputs: (&Vector<f32>, &Vector<f32>) = (&v1, &v2);
            group.bench_with_input(
                BenchmarkId::new("scalar_dot", size),
                &inputs,
                |b, (v1, v2)| b.iter(|| v1.dot(v2)),
            );
        }

        {
            let inputs: (&Vector<f32>, &Vector<f32>) = (&v1, &v2);
            group.bench_with_input(
                BenchmarkId::new("simd_dot", size),
                &inputs,
                |b, (v1, v2)| b.iter(|| unsafe { dot_simd(*v1, *v2) }),
            );
        }
    }

    group.finish();
}
