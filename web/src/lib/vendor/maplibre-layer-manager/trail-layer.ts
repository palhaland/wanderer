import type { FilterSpecification, MapMouseEvent, Marker, StyleSpecification } from "maplibre-gl";
import type { BaseLayer } from "./layers";

export class TrailLayer implements BaseLayer {

    spec: StyleSpecification;
    listeners: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }>
    markers: Record<string, Marker> = {};

    constructor(id: string, geojson: GeoJSON.FeatureCollection, color: string, options?: {
        minZoom?: number,
        maxZoom?: number,
        tiers?: { thresholds: number[], limits: number[] },
        listeners?: { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }
    }) {
        const tiers = options?.tiers;
        let filter: FilterSpecification | undefined = undefined;

        if (tiers) {
            const pairs = tiers.thresholds.map((t, i) => ({
                threshold: t,
                limit: tiers.limits[i]
            })).sort((a, b) => a.threshold - b.threshold);

            const thresholds: number[] = [];
            const limits: number[] = [];
            let lastT = -1;
            for (const p of pairs) {
                if (p.threshold > lastT) {
                    thresholds.push(p.threshold);
                    limits.push(p.limit);
                    lastT = p.threshold;
                }
            }

            const stepExpr: any[] = ["step", ["zoom"], limits[0]];
            for (let i = 0; i < thresholds.length; i++) {
                stepExpr.push(thresholds[i], limits[i + 1] ?? 0);
            }

            filter = [
                ">",
                ["get", "bounding_box_diagonal"],
                stepExpr as any
            ];
        }

        const layer: M.LineLayerSpecification = {
            id: id,
            type: "line",
            source: id,
            paint: {
                "line-color": color,
                "line-width": 5,
            },
        };
        
        if (filter !== undefined) layer.filter = filter;
        if (options?.maxZoom !== undefined) layer.maxzoom = options.maxZoom;

        this.spec = {
            version: 8,
            name: id,
            sources: {
                [id]: {
                    type: "geojson",
                    data: geojson,
                }
            },
            layers: [layer]

        };

        this.listeners = { [id]: options?.listeners ?? {} }
    }
}