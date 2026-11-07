import { notFound } from "next/navigation";
import { getProvider } from "@/lib/admin-api";
import { ProviderDetail } from "./provider-detail";

export default async function ProviderDetailPage({
  params
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const provider = await getProvider(id);
  if (!provider) notFound();
  return <ProviderDetail provider={provider} />;
}
