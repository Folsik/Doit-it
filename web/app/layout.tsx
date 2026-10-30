import type { Metadata, Viewport } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Doit — Money, made clearer",
  description: "A modern personal finance experience.",
  applicationName: "Doit",
};
export const viewport: Viewport = { width: "device-width", initialScale: 1, themeColor: "#f5f7fb" };

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>;
}
