export interface SavedRouteSummary {
  id: string;
  status: string;
  source: string;
  createdAt: string;
  name?: string;
}

export interface SavedRouteDetails extends SavedRouteSummary {
  orderedTaskIds: string[];
  totalDistanceKm?: number;
  totalTravelTimeMin?: number;
  totalDurationMin?: number;
}
