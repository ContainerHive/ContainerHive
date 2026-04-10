export interface SBOMPackage {
  name: string;
  version?: string;
}

export interface PlatformReport {
  platform: string;
  digest: string;
  size: number;
  hasSbom: boolean;
  sbom?: SBOMPackage[];
  buildArgs?: Record<string, string>;
}

export interface TagReport {
  name: string;
  versions?: Record<string, string>;
  platforms: PlatformReport[];
}

export interface VariantReport {
  name: string;
  icon?: string;
  aliases?: string[];
  tagSuffix: string;
  tags: TagReport[];
}

export interface ImageReport {
  name: string;
  icon?: string;
  versions?: Record<string, string>;
  aliases?: string[];
  tags: TagReport[];
  variants?: VariantReport[];
}

export interface ProjectReport {
  generatedAt: string;
  source: string;
  registryAddr?: string;
  images: ImageReport[];
}