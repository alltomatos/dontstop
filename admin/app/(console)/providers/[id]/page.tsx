import { notFound } from "next/navigation";
import { PROVIDERS } from "@/lib/providers";
import { ProviderDetail } from "./provider-detail";

export default async function ProviderDetailPage({
  params
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const provider = PROVIDERS[id];
  if (!provider) notFound();
  return <ProviderDetail provider={provider} />;
}
