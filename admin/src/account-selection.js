(function (root, factory) {
  if (typeof module === "object" && module.exports) module.exports = factory();
  if (root) root.DontstopAccountSelection = factory();
})(typeof globalThis !== "undefined" ? globalThis : this, function () {
  function eligibleAccounts(accounts) {
    return (Array.isArray(accounts) ? accounts : []).filter(
      (account) => account && account.id && account.active === true && Number(account.modelsUsable || 0) > 0
    );
  }

  function pickTestAccount(accounts, savedID) {
    const eligible = eligibleAccounts(accounts);
    if (savedID) {
      const saved = eligible.find((account) => account.id === savedID);
      if (saved) return saved;
    }
    return eligible[0];
  }

  function rememberTestAccount(store, providerID, accountID) {
    return { ...(store && typeof store === "object" ? store : {}), [providerID]: accountID };
  }

  return { eligibleAccounts, pickTestAccount, rememberTestAccount };
});
