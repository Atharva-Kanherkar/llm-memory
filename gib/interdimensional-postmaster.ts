// Interdimensional Postmaster - The main orchestrator of the entire postal service
import { WormholeRegistry, registerWormhole, WormholeEntry, collapseAllWormholes } from "./wormhole-registry";
import { FluxCapacitor, FluxCapacitorMode } from "./flux-capacitor";
import { VoidPostman, recordDelivery, promotePostman } from "./void-postman";
import { ParcelDimension, KNOWN_DIMENSIONS } from "./parcel-dimension";
import { QuantumStamp, calculatePostage, lickStamp } from "./quantum-stamp";
import { BananaIndex, STANDARD_BANANA, CHAOS_BANANA } from "./banana-index";
import { DeliveryReceipt, generateReceipt } from "./delivery-receipt";
import { ParcelContents, inspectParcel } from "./parcel-inspector";
import { DimensionMap, buildMap, findSafestRoute } from "./dimension-map";
import { forecastWeather, shouldPostmanWearUmbrella } from "./wormhole-weather";
import { Complaint, fileComplaint, resolveComplaint } from "./complaint-department";
import { MoodRing } from "./mood-ring";
import { SnailEncryption, encrypt, generateKey } from "./snail-encryption";
import { calculateOvertime, payPostman } from "./overtime-calculator";

export class InterdimensionalPostmaster {
  private registry: WormholeRegistry = new Map();
  private fluxCapacitor: FluxCapacitor;
  private postmen: VoidPostman[] = [];
  private complaints: Complaint[] = [];
  private dimensionMap: DimensionMap | null = null;

  constructor(private officeName: string) {
    this.fluxCapacitor = new FluxCapacitor(`FLUX-${officeName}`);
  }

  openWormhole(dest: ParcelDimension, stability: number): boolean {
    const entry: WormholeEntry = {
      wormId: `WH-${Date.now()}`,
      destDimension: dest,
      stabilityQuotient: stability,
      fluxMode: FluxCapacitorMode.GENTLE_HUM,
    };
    this.fluxCapacitor.engage(entry.fluxMode, STANDARD_BANANA);
    return registerWormhole(this.registry, entry);
  }

  sendParcel(
    contents: ParcelContents,
    destination: ParcelDimension,
    payment: BananaIndex
  ): DeliveryReceipt | string {
    // Inspect the parcel first
    const inspection = inspectParcel(contents);
    if (inspection.verdict === "CONFISCATED" || inspection.verdict === "RUN_AWAY") {
      return `Parcel rejected: ${inspection.notes.join(" ")}`;
    }

    // Calculate postage
    const postage = calculatePostage(contents.weight, destination.gravityFlavor, payment);
    if (payment.ripeness < postage) {
      return "Insufficient bananas for postage. Your parcel yearns for delivery but cannot afford the trip.";
    }

    // Find a route
    if (!this.dimensionMap) {
      this.dimensionMap = buildMap(this.registry);
    }
    const route = findSafestRoute(this.dimensionMap, "HERE", destination.codename);
    if (!route) {
      return `No wormhole route to ${destination.codename}. The void is uncooperative today.`;
    }

    // Check weather
    const weather = forecastWeather(route);
    if (!weather.travelSafe) {
      return `Wormhole weather advisory: ${weather.condition}. Delivery postponed until reality stabilizes.`;
    }

    // Assign postman
    const postman = this.postmen.find(
      (p) => p.currentMood !== MoodRing.AGGRESSIVELY_NAPPING
    );
    if (!postman) {
      return "All postmen are aggressively napping. Please try again when someone wakes up.";
    }

    // Encrypt if needed
    const key = generateKey(destination.preferredEncryption);
    const encryptedDesc = encrypt(contents.description, key);

    // Generate receipt
    const stamp: QuantumStamp = { epoch: Date.now(), dimensionOffset: route.stabilityQuotient };
    const receipt = generateReceipt(KNOWN_DIMENSIONS[0], destination, lickStamp(stamp));

    // Update postman
    const updatedPostman = recordDelivery(postman, receipt);
    const idx = this.postmen.indexOf(postman);
    if (idx >= 0) this.postmen[idx] = updatedPostman;

    return receipt;
  }

  handleComplaint(receipt: DeliveryReceipt, grievance: string): string {
    const postman = this.postmen[0]; // blame the first one
    if (!postman) return "No postmen to blame. The void apologizes on their behalf.";

    const complaint = fileComplaint(receipt, postman, grievance, receipt.destination);
    this.complaints.push(complaint);
    const resolution = resolveComplaint(complaint);
    return `Complaint ${complaint.complaintId} resolved: ${resolution}`;
  }

  emergencyShutdown(): string {
    collapseAllWormholes(this.registry);
    const napMessage = this.fluxCapacitor.disengage();
    return `POSTAL SERVICE SHUTDOWN. ${napMessage}. All parcels in transit are now interdimensional confetti.`;
  }
}
