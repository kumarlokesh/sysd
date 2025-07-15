use crate::vector::Vector;
use std::arch::x86_64::*;

/// SIMD implementation of vector operations

/// SIMD vector addition
pub fn add_simd(v1: &Vector<f32>, v2: &Vector<f32>, result: &mut Vector<f32>) {
    assert_eq!(v1.len(), v2.len(), "Vector lengths must match");
    assert_eq!(v1.len(), result.len(), "Result vector length must match");
    assert!(
        v1.len() % 8 == 0,
        "Vector length must be multiple of 8 for SIMD"
    );

    let len = v1.len();
    let chunks = len / 8;

    unsafe {
        for i in 0..chunks {
            let offset = i * 8;
            let a_ptr = v1.as_slice().as_ptr().add(offset);
            let b_ptr = v2.as_slice().as_ptr().add(offset);
            let result_ptr = result.as_mut_slice().as_mut_ptr().add(offset);

            let av = _mm256_loadu_ps(a_ptr);
            let bv = _mm256_loadu_ps(b_ptr);
            let sum = _mm256_add_ps(av, bv);
            _mm256_storeu_ps(result_ptr, sum);
        }
    }
}

/// SIMD vector multiplication
pub fn mul_simd(v1: &Vector<f32>, v2: &Vector<f32>, result: &mut Vector<f32>) {
    assert_eq!(v1.len(), v2.len(), "Vector lengths must match");
    assert_eq!(v1.len(), result.len(), "Result vector length must match");
    assert!(
        v1.len() % 8 == 0,
        "Vector length must be multiple of 8 for SIMD"
    );

    let len = v1.len();
    let chunks = len / 8;

    unsafe {
        for i in 0..chunks {
            let offset = i * 8;
            let a_ptr = v1.as_slice().as_ptr().add(offset);
            let b_ptr = v2.as_slice().as_ptr().add(offset);
            let result_ptr = result.as_mut_slice().as_mut_ptr().add(offset);

            let av = _mm256_loadu_ps(a_ptr);
            let bv = _mm256_loadu_ps(b_ptr);
            let prod = _mm256_mul_ps(av, bv);
            _mm256_storeu_ps(result_ptr, prod);
        }
    }
}

/// SIMD dot product
pub unsafe fn dot_simd(v1: &Vector<f32>, v2: &Vector<f32>) -> f32 {
    assert_eq!(v1.len(), v2.len(), "Vector lengths must match");
    assert!(
        v1.len() % 8 == 0,
        "Vector length must be multiple of 8 for SIMD"
    );

    let len = v1.len();
    let chunks = len / 8;

    unsafe {
        let mut sum = _mm256_setzero_ps();
        for i in 0..chunks {
            let offset = i * 8;
            let a_ptr = v1.as_slice().as_ptr().add(offset);
            let b_ptr = v2.as_slice().as_ptr().add(offset);

            let av = _mm256_loadu_ps(a_ptr);
            let bv = _mm256_loadu_ps(b_ptr);
            let prod = _mm256_mul_ps(av, bv);
            sum = _mm256_add_ps(sum, prod);
        }

        // Horizontal sum
        let mut tmp = _mm256_extractf128_ps(sum, 1);
        tmp = _mm_add_ps(tmp, _mm256_extractf128_ps(sum, 0));
        tmp = _mm_add_ps(tmp, _mm_permute_ps(tmp, 0x55));
        tmp = _mm_add_ps(tmp, _mm_permute_ps(tmp, 0x39));
        _mm_cvtss_f32(tmp)
    }
}

/// Helper function to create SIMD-compatible vectors
pub fn create_simd_vector(length: usize, value: f32) -> Vector<f32> {
    // Round up to nearest multiple of 8 for AVX2
    let padded_length = (length + 7) / 8 * 8;
    Vector::with_length(padded_length, value)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_simd_addition() {
        let v1 = create_simd_vector(8, 1.0);
        let v2 = create_simd_vector(8, 2.0);
        let mut result = create_simd_vector(8, 0.0);
        add_simd(&v1, &v2, &mut result);
        assert_eq!(result.as_slice(), &[3.0; 8]);
    }

    #[test]
    fn test_simd_multiplication() {
        let v1 = create_simd_vector(8, 2.0);
        let v2 = create_simd_vector(8, 3.0);
        let mut result = create_simd_vector(8, 0.0);
        mul_simd(&v1, &v2, &mut result);
        assert_eq!(result.as_slice(), &[6.0; 8]);
    }

    #[test]
    fn test_simd_dot_product() {
        let v1 = create_simd_vector(8, 2.0);
        let v2 = create_simd_vector(8, 3.0);
        let result = unsafe { dot_simd(&v1, &v2) };
        assert_eq!(result, 48.0); // 2 * 3 * 8 = 48
    }
}
