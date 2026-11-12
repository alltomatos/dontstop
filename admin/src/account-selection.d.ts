export interface TestableAccount {
  id?: string;
  name: string;
  active?: boolean;
  health?: string;
  modelsUsable?: number;
}

export function eligibleAccounts<T extends TestableAccount>(accounts: ReadonlyArray<T>): T[];
export function pickTestAccount<T extends TestableAccount>(
  accounts: ReadonlyArray<T>,
  savedID?: string
): T | undefined;
export function rememberTestAccount(
  store: Record<string, string>,
  providerID: string,
  accountID: string
): Record<string, string>;
