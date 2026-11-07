import type { Metadata } from "next";
import "./globals.css";
import { I18nProvider } from "@/components/console/i18n-provider";

export const metadata: Metadata = {
  title: "DonStop IA — Console",
  description: "Console administrativo do gateway corporativo DonStop IA."
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <I18nProvider>{children}</I18nProvider>
      </body>
    </html>
  );
}
