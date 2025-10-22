module.exports = {
  testEnvironment: "jsdom",
  verbose: true,
  setupFilesAfterEnv: ["<rootDir>/test/jest.setup.cjs"],
  testPathIgnorePatterns: ["<rootDir>/ui-tests/"],
};
