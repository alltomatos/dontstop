import { getComboCatalogResult, getCombosResult } from "@/lib/combos";
import { CombosView } from "./combos-view";

export default async function CombosPage() {
  const [{ combos, error }, catalog] = await Promise.all([getCombosResult(), getComboCatalogResult()]);
  return (
    <CombosView
      combos={combos}
      models={catalog.models}
      error={error || catalog.error}
    />
  );
}
