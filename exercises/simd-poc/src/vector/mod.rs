use std::fmt;
use std::ops::Add;

/// A generic vector type for SIMD operations
#[derive(Debug, Clone)]
pub struct Vector<T> {
    data: Vec<T>,
}

impl<T: Copy> Vector<T> {
    /// Create a new vector with a specific length and default value
    pub fn with_length(length: usize, value: T) -> Self {
        Self {
            data: vec![value; length],
        }
    }

    /// Get the length of the vector
    pub fn len(&self) -> usize {
        self.data.len()
    }

    /// Get a reference to the underlying data
    pub fn as_slice(&self) -> &[T] {
        &self.data
    }

    /// Get a mutable reference to the underlying data
    pub fn as_mut_slice(&mut self) -> &mut [T] {
        &mut self.data
    }
}

impl<T: Copy + Add<Output = T>> Add for Vector<T> {
    type Output = Self;

    /// Scalar addition of two vectors
    fn add(self, other: Self) -> Self {
        assert_eq!(self.len(), other.len(), "Vector lengths must match");
        let mut result = self.clone();
        for i in 0..self.len() {
            result.data[i] = self.data[i] + other.data[i];
        }
        result
    }
}

impl Vector<f32> {
    /// Compute the dot product of two vectors
    pub fn dot(&self, other: &Self) -> f32 {
        assert_eq!(self.len(), other.len(), "Vector lengths must match");
        self.data
            .iter()
            .zip(other.data.iter())
            .map(|(a, b)| a * b)
            .sum()
    }

    /// Generate a random vector with values between min and max
    pub fn random(length: usize, min: f32, max: f32) -> Self {
        use rand::Rng;
        let mut rng = rand::thread_rng();
        Self {
            data: (0..length).map(|_| rng.gen_range(min..=max)).collect(),
        }
    }
}

impl<T: std::ops::Mul<Output = T> + Copy> std::ops::Mul for Vector<T> {
    type Output = Self;

    fn mul(self, other: Self) -> Self {
        assert_eq!(self.len(), other.len(), "Vector lengths must match");
        Self {
            data: self
                .data
                .iter()
                .zip(other.data.iter())
                .map(|(a, b)| *a * *b)
                .collect(),
        }
    }
}

impl<T: fmt::Display> fmt::Display for Vector<T> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "[")?;
        for (i, item) in self.data.iter().enumerate() {
            if i > 0 {
                write!(f, ", ")?;
            }
            write!(f, "{}", item)?;
        }
        write!(f, "]")
    }
}

pub mod simd;

pub use simd::{add_simd, create_simd_vector, dot_simd, mul_simd};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_vector_addition() {
        let v1 = Vector::with_length(3, 1.0);
        let v2 = Vector::with_length(3, 2.0);
        let result = v1 + v2;
        assert_eq!(result.as_slice(), &[3.0, 3.0, 3.0]);
    }

    #[test]
    fn test_vector_multiplication() {
        let v1 = Vector::with_length(3, 2.0);
        let v2 = Vector::with_length(3, 3.0);
        let result = v1 * v2;
        assert_eq!(result.as_slice(), &[6.0, 6.0, 6.0]);
    }

    #[test]
    fn test_dot_product() {
        let v1 = Vector::with_length(3, 2.0);
        let v2 = Vector::with_length(3, 3.0);
        assert_eq!(v1.dot(&v2), 18.0);
    }

    #[test]
    fn test_random_vector() {
        let v = Vector::random(3, 0.0, 10.0);
        assert_eq!(v.len(), 3);
        assert!(v.as_slice().iter().all(|&x| x >= 0.0 && x <= 10.0));
    }
}
