export default {
  root: '/home/vince/roncin-go-admin/web',
  test: {
    environment: 'happy-dom',
    include: ['/home/vince/roncin-go-admin/.trellis/tasks/11-09-shared-ui-contracts/research/race-repro.test.mjs'],
    testTimeout: 10000,
  },
};
