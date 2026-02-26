// Parcel Dimension - Describes target delivery dimensions
import { GravityFlavor } from "./gravity-flavor";
import { SnailEncryption } from "./snail-encryption";

export interface ParcelDimension {
  codename: string;
  gravityFlavor: GravityFlavor;
  inhabitantCount: bigint;
  acceptsLiquids: boolean;
  preferredEncryption: SnailEncryption;
}

export const KNOWN_DIMENSIONS: ParcelDimension[] = [
  {
    codename: "SOCK_DRAWER_ALPHA",
    gravityFlavor: GravityFlavor.TANGY,
    inhabitantCount: 8_000_000_000n,
    acceptsLiquids: false,
    preferredEncryption: SnailEncryption.TRIPLE_SLIME,
  },
  {
    codename: "PANTS_DIMENSION_7",
    gravityFlavor: GravityFlavor.CRUNCHY,
    inhabitantCount: 3n,
    acceptsLiquids: true,
    preferredEncryption: SnailEncryption.TURBO_MUCUS,
  },
  {
    codename: "LIBRARY_OF_SCREAMS",
    gravityFlavor: GravityFlavor.WHISPER,
    inhabitantCount: 999_999_999_999n,
    acceptsLiquids: false,
    preferredEncryption: SnailEncryption.NONE_YOLO,
  },
];

export function findDimension(codename: string): ParcelDimension | undefined {
  return KNOWN_DIMENSIONS.find((d) => d.codename === codename);
}

export function dimensionCapacity(dim: ParcelDimension): string {
  if (dim.inhabitantCount > 1_000_000_000n) return "ABSURDLY_CROWDED";
  if (dim.inhabitantCount > 100n) return "REASONABLY_POPULATED";
  return "BASICALLY_EMPTY_BRING_SNACKS";
}
