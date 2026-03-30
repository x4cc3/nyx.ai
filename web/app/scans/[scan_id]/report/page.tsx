export const dynamic = "force-dynamic";

import type { Metadata } from "next";
import ReportClient from "./ReportClient";

interface ReportPageProps {
  params: Promise<{ scan_id: string }>;
}

export async function generateMetadata({ params }: ReportPageProps): Promise<Metadata> {
  const { scan_id } = await params;
  return {
    title: `Report – Scan ${scan_id.slice(0, 8)} – NYX`,
    description: `Security audit report for scan ${scan_id}`,
  };
}

export default async function ReportPage({ params }: ReportPageProps) {
  const { scan_id: scanId } = await params;
  return <ReportClient scanId={scanId} />;
}
