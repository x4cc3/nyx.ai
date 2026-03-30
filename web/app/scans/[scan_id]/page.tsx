export const dynamic = "force-dynamic";

import type { Metadata } from "next";
import LiveScanClient from "./LiveScanClient";

interface LiveScanPageProps {
  params: Promise<{ scan_id: string }>;
}

export async function generateMetadata({ params }: LiveScanPageProps): Promise<Metadata> {
  const { scan_id } = await params;
  return {
    title: `Scan ${scan_id.slice(0, 8)} – NYX`,
    description: `Live scan view for ${scan_id}`,
  };
}

export default async function LiveScanPage({ params }: LiveScanPageProps) {
  const { scan_id: scanId } = await params;
  return <LiveScanClient scanId={scanId} />;
}
