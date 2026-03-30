import { requireSession } from "@/lib/serverAuth";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export default async function ScansLayout({ children }: { children: React.ReactNode }) {
  await requireSession("/scans");
  return children;
}
