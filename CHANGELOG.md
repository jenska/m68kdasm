# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-03-28

### Fixed
- **MOVEM register list decoding**: Fixed incorrect register mask interpretation for Mem→Reg operations. Previously `4C DF 0C 04` decoded as `MOVEM.L (A7)+, A4-A5/D5` instead of the correct `MOVEM.L (A7)+, D2/A2-A3`.
- **SWAP instruction decoding**: Added proper SWAP instruction decoder to distinguish from PEA. Previously `48 40` decoded as `PEA D0` instead of `SWAP D0`.
- **Code cleanup**: Removed unused direction parameter from `formatRegisterList` function and added documentation for new constants.

### Added
- Comprehensive test cases for MOVEM, SWAP, and branch instructions to prevent regressions.

## [1.0.0] - 2026-03-XX

Initial release of m68kdasm, a Go disassembler for the Motorola 68000 CPU.

### Features
- Fast opcode dispatch using hierarchical jump table
- Broad 68000 instruction coverage
- Full addressing mode decoding
- Structured metadata output
- Multiple decode entry points (slice, io.ReaderAt, callback)
- ELF binary support
- Precise error reporting