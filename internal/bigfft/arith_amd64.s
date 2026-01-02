// Copyright 2024 The fibcalc Authors.
// This file provides optimized arithmetic operations for big integer FFT.
//
// These implementations use hand-optimized x86-64 assembly with loop unrolling
// for maximum performance on large vectors.
//
// Note on AVX2: While the function names include "_avx2" for API compatibility,
// vector addition/subtraction cannot efficiently use AVX2 due to the lack of
// native carry/borrow propagation across 64-bit lanes. The multiply-accumulate
// function benefits from careful register allocation and loop unrolling instead.

#include "textflag.h"

// ─────────────────────────────────────────────────────────────────────────────
// addVV_avx2: z = x + y with carry propagation
// ─────────────────────────────────────────────────────────────────────────────
//
// func addVVAvx2(z, x, y []Word) (c Word)
//
// Adds two vectors x and y, storing result in z. Returns the final carry.
//
// Implementation note: AVX2's VPADDQ instruction cannot propagate carries
// across 64-bit lanes, so we use an optimized scalar implementation with
// the ADC (add with carry) instruction chain.

TEXT ·addVVAvx2(SB), NOSPLIT, $0-80
    // Arguments:
    // z []Word: FP+0 (ptr), FP+8 (len), FP+16 (cap)
    // x []Word: FP+24 (ptr), FP+32 (len), FP+40 (cap)
    // y []Word: FP+48 (ptr), FP+56 (len), FP+64 (cap)
    // return c: FP+72

    MOVQ z_base+0(FP), DI      // DI = &z[0]
    MOVQ x_base+24(FP), SI     // SI = &x[0]
    MOVQ y_base+48(FP), DX     // DX = &y[0]
    MOVQ z_len+8(FP), CX       // CX = len(z)

    XORQ AX, AX                // AX = carry = 0

    // Check for empty input
    TESTQ CX, CX
    JZ    addvv_done

addvv_loop:
    MOVQ (SI), R8              // R8 = x[i]
    ADDQ AX, R8                // R8 = x[i] + carry_in
    MOVQ $0, AX                // Reset carry
    ADCQ $0, AX                // AX = carry from previous add
    ADDQ (DX), R8              // R8 = x[i] + carry_in + y[i]
    ADCQ $0, AX                // AX += carry from this add
    MOVQ R8, (DI)              // z[i] = result

    ADDQ $8, SI                // x++
    ADDQ $8, DX                // y++
    ADDQ $8, DI                // z++
    DECQ CX                    // len--
    JNZ  addvv_loop

addvv_done:
    MOVQ AX, c+72(FP)          // return carry
    RET

// ─────────────────────────────────────────────────────────────────────────────
// subVV_avx2: z = x - y with borrow propagation
// ─────────────────────────────────────────────────────────────────────────────
//
// func subVVAvx2(z, x, y []Word) (c Word)
//
// Subtracts y from x, storing result in z. Returns the final borrow.
// Uses optimized SBB (subtract with borrow) chain.

TEXT ·subVVAvx2(SB), NOSPLIT, $0-80
    MOVQ z_base+0(FP), DI      // DI = &z[0]
    MOVQ x_base+24(FP), SI     // SI = &x[0]
    MOVQ y_base+48(FP), DX     // DX = &y[0]
    MOVQ z_len+8(FP), CX       // CX = len(z)

    XORQ AX, AX                // AX = borrow = 0

    // Check for empty input
    TESTQ CX, CX
    JZ    subvv_done

subvv_loop:
    MOVQ (SI), R8              // R8 = x[i]
    MOVQ (DX), R9              // R9 = y[i]
    
    // Subtract borrow from x[i]
    SUBQ AX, R8                // R8 = x[i] - borrow_in
    SBBQ AX, AX                // AX = -1 if borrow, 0 otherwise
    NEGQ AX                    // AX = 1 if borrow, 0 otherwise
    
    // Subtract y[i]
    SUBQ R9, R8                // R8 = x[i] - borrow_in - y[i]
    SBBQ R9, R9                // R9 = -1 if borrow, 0 otherwise
    NEGQ R9                    // R9 = 1 if borrow, 0 otherwise
    
    // Combine borrows (at most one can be set)
    ORQ R9, AX                 // AX = total borrow
    MOVQ R8, (DI)              // z[i] = result

    ADDQ $8, SI                // x++
    ADDQ $8, DX                // y++
    ADDQ $8, DI                // z++
    DECQ CX                    // len--
    JNZ  subvv_loop

subvv_done:
    MOVQ AX, c+72(FP)          // return borrow
    RET

// ─────────────────────────────────────────────────────────────────────────────
// addMulVVW_avx2: z += x * y where y is a single word
// ─────────────────────────────────────────────────────────────────────────────
//
// func addMulVVWAvx2(z, x []Word, y Word) (c Word)
//
// Computes z[i] += x[i] * y for all i, propagating carry.
// This is the CRITICAL inner loop for basicMul in FFT multiplication.
//
// Optimization: 4x loop unrolling reduces loop overhead by 75% and enables
// better instruction scheduling. Each iteration processes 4 words (32 bytes).
//
// Register allocation:
//   DI = z pointer
//   SI = x pointer
//   R8 = y (multiplier, preserved across MULQ)
//   CX = loop counter
//   BX = carry
//   AX, DX = used by MULQ (DX:AX = result)

TEXT ·addMulVVWAvx2(SB), NOSPLIT, $0-64
    // Arguments:
    // z []Word: FP+0 (ptr), FP+8 (len), FP+16 (cap)
    // x []Word: FP+24 (ptr), FP+32 (len), FP+40 (cap)
    // y Word: FP+48
    // return c: FP+56

    MOVQ z_base+0(FP), DI      // DI = &z[0]
    MOVQ x_base+24(FP), SI     // SI = &x[0]
    MOVQ y+48(FP), R8          // R8 = y (multiplier)
    MOVQ z_len+8(FP), CX       // CX = len(z)

    XORQ BX, BX                // BX = carry = 0

    // Check for empty input
    TESTQ CX, CX
    JZ    addmulvvw_done

    // Check if we have at least 4 words for unrolled loop
    CMPQ CX, $4
    JL   addmulvvw_tail

addmulvvw_unroll4:
    // ─── Word 0 ───
    MOVQ (SI), AX              // AX = x[0]
    MULQ R8                    // DX:AX = x[0] * y
    ADDQ BX, AX                // lo += carry
    ADCQ $0, DX                // hi += overflow
    ADDQ (DI), AX              // lo += z[0]
    ADCQ $0, DX                // hi += overflow
    MOVQ AX, (DI)              // z[0] = lo
    MOVQ DX, BX                // carry = hi

    // ─── Word 1 ───
    MOVQ 8(SI), AX             // AX = x[1]
    MULQ R8                    // DX:AX = x[1] * y
    ADDQ BX, AX                // lo += carry
    ADCQ $0, DX                // hi += overflow
    ADDQ 8(DI), AX             // lo += z[1]
    ADCQ $0, DX                // hi += overflow
    MOVQ AX, 8(DI)             // z[1] = lo
    MOVQ DX, BX                // carry = hi

    // ─── Word 2 ───
    MOVQ 16(SI), AX            // AX = x[2]
    MULQ R8                    // DX:AX = x[2] * y
    ADDQ BX, AX                // lo += carry
    ADCQ $0, DX                // hi += overflow
    ADDQ 16(DI), AX            // lo += z[2]
    ADCQ $0, DX                // hi += overflow
    MOVQ AX, 16(DI)            // z[2] = lo
    MOVQ DX, BX                // carry = hi

    // ─── Word 3 ───
    MOVQ 24(SI), AX            // AX = x[3]
    MULQ R8                    // DX:AX = x[3] * y
    ADDQ BX, AX                // lo += carry
    ADCQ $0, DX                // hi += overflow
    ADDQ 24(DI), AX            // lo += z[3]
    ADCQ $0, DX                // hi += overflow
    MOVQ AX, 24(DI)            // z[3] = lo
    MOVQ DX, BX                // carry = hi

    // Advance pointers and counter
    ADDQ $32, SI               // x += 4
    ADDQ $32, DI               // z += 4
    SUBQ $4, CX                // len -= 4
    
    CMPQ CX, $4
    JGE  addmulvvw_unroll4

addmulvvw_tail:
    // Handle remaining 0-3 words
    TESTQ CX, CX
    JZ    addmulvvw_done

addmulvvw_loop:
    MOVQ (SI), AX              // AX = x[i]
    MULQ R8                    // DX:AX = x[i] * y
    ADDQ BX, AX                // lo += carry
    ADCQ $0, DX                // hi += overflow
    ADDQ (DI), AX              // lo += z[i]
    ADCQ $0, DX                // hi += overflow
    MOVQ AX, (DI)              // z[i] = lo
    MOVQ DX, BX                // carry = hi

    ADDQ $8, SI                // x++
    ADDQ $8, DI                // z++
    DECQ CX                    // len--
    JNZ  addmulvvw_loop

addmulvvw_done:
    MOVQ BX, c+56(FP)          // return carry
    RET
