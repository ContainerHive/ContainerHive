export interface SBOMPackage {
  name: string;
  version?: string;
}

export interface TagReport {
  name: string;
  buildArgs?: Record<string, string>;
  versions?: Record<string, string>;
  platforms?: PlatformReport[];
}

export interface PlatformReport {
  platform: string;
  sbom?: SBOMPackage[];
}

export interface VariantReport {
  name: string;
  readme?: string;
  report?: { icon?: string };
  tagSuffix: string;
  platforms?: string[];
  tags: TagReport[];
}

export interface ImageReport {
  name: string;
  description?: string;
  readme?: string;
  report?: { icon?: string };
  platforms?: string[];
  tags: TagReport[];
  variants?: VariantReport[];
}

export interface RegistryInfo {
  address: string;
}

export interface ProjectReport {
  generatedAt: string;
  registry?: RegistryInfo;
  source: string;
  images: ImageReport[];
}