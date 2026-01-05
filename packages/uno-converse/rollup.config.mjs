import typescript from '@rollup/plugin-typescript';
import dts from 'rollup-plugin-dts';

const external = ['react', 'react/jsx-runtime'];

export default [
  // ESM and CJS builds
  {
    input: 'src/index.ts',
    output: [
      {
        file: 'dist/index.js',
        format: 'esm',
        sourcemap: true,
      },
      {
        file: 'dist/index.cjs',
        format: 'cjs',
        sourcemap: true,
      },
    ],
    external,
    plugins: [
      typescript({
        tsconfig: './tsconfig.json',
        declaration: false,
        noEmit: false,
        outDir: './dist',
      }),
    ],
  },
  // Type declarations
  {
    input: 'src/index.ts',
    output: [
      { file: 'dist/index.d.ts', format: 'esm' },
      { file: 'dist/index.d.cts', format: 'cjs' },
    ],
    external,
    plugins: [dts()],
  },
];

