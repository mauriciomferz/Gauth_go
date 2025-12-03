// Example Jest test for tokens.js
describe("tokens module", () => {
  it("should initialize without error", async () => {
    const mod = await import("../modules/tokens.js");
    expect(() => mod.initTokens()).not.toThrow();
  });
});
