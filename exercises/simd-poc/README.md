# SIMD Proof-of-Concept

This project demonstrates practical SIMD (Single Instruction Multiple Data) optimization techniques using Rust's SIMD capabilities. It provides a hands-on understanding of SIMD implementation and performance benefits.

## Requirements

- Rust 1.64+ (for stable SIMD support)
- x86_64 architecture (for AVX/AVX2 support)

## Usage

```bash
# Build the project
cargo build --release

# Run benchmarks
cargo run --release --features="simd"
```

For detailed technical design and implementation details, see [DESIGN.md](DESIGN.md).
