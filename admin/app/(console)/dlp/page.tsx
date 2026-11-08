import { getDlpEvents, getDlpRules } from "@/lib/admin-api";
import { DlpView } from "./dlp-view";

export default async function DlpPage() {
  const [rules, events] = await Promise.all([getDlpRules(), getDlpEvents(20)]);
  return <DlpView rules={rules} events={events} />;
}
