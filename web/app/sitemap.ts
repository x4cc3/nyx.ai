import type { MetadataRoute } from "next";
import { getSiteOrigin } from "@/lib/env";

export default function sitemap(): MetadataRoute.Sitemap {
  const site = getSiteOrigin();
  return [
    {
      url: `${site}/`,
      changeFrequency: "daily",
      priority: 1,
    },
    {
      url: `${site}/login`,
      changeFrequency: "monthly",
      priority: 0.8,
    },
    {
      url: `${site}/register`,
      changeFrequency: "monthly",
      priority: 0.8,
    },
  ];
}
