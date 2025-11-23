import { createMDXSource } from "fumadocs-mdx"
import { loader } from "fumadocs-core/source"
import { docs } from "@/source.config"

export function getSource() {
  return loader({
    baseUrl: "/docs",
    source: createMDXSource(docs, {}),
  })
}