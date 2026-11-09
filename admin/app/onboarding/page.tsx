import { redirect } from "next/navigation";
import { getSetupStatus } from "@/lib/admin-api";
import { OnboardingWizard } from "./onboarding-wizard";

export const dynamic = "force-dynamic";

export default async function OnboardingPage() {
  if (!(await getSetupStatus())) redirect("/login");
  return (
    <main className="login-wrap">
      <OnboardingWizard />
    </main>
  );
}
