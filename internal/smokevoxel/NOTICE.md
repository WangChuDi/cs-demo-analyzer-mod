The decoder and density simulator in `journal.go` and `simulation.go` are Go
adaptations of `src/helpers/smokeVoxel.ts` and `src/helpers/smokeDensity.ts` in
[osztenkurden/cs2parser](https://github.com/osztenkurden/cs2parser), pinned to
commit `0c6c22e95251dfd6432d470ee3bbd3be2d61173d`, under MPL-2.0. The native-oracle
test data is adapted from that revision's `smoke-voxels.json` and
`smoke-density.json`. Their license is included as `LICENSE-MPL-2.0`.

This is an experimental reconstruction of raw simulation density, not rendered
smoke opacity or a direct player-in-smoke flag. The upstream oracle identifies
Linux client SHA-256
`708200eadba165d45fdeb7b7eaa67a3fd5f6752097f723b8cb435a33d73b462a`.
The local Go port is compared against the published oracle hashes; no native
client or pixel comparison has been performed in this workspace.
