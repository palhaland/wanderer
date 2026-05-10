import type { MapMouseEvent, StyleSpecification } from "maplibre-gl";
import type { BaseLayer } from "./layers";
import * as M from "maplibre-gl";

export class PreviewLayer implements BaseLayer {

    private map: M.Map;

    spec: StyleSpecification;
    listeners: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }> = {
        "preview": {
            onEnter: () => this.map!.getCanvas().style.cursor = "pointer",
            onLeave: () => this.map!.getCanvas().style.cursor = ""
        },
        "preview-start-points": {
            onEnter: () => this.map!.getCanvas().style.cursor = "pointer",
            onLeave: () => this.map!.getCanvas().style.cursor = ""
        }
    };

    constructor(map: M.Map, geojson: GeoJSON.FeatureCollection, minZoom: number = 10, options?: { tiers?: { thresholds: number[], limits: number[] }, listeners?: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }> }) {

        this.map = map;
        const listeners = options?.listeners;
        this.listeners = {
            "preview": { ...this.listeners["preview"], ...listeners?.["preview"] },
            "preview-start-points": { ...this.listeners["preview-start-points"], ...listeners?.["preview-start-points"] }
        }

        const tiers = options?.tiers;
        let lineFilter: M.FilterSpecification | undefined = undefined;

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

            lineFilter = [
                ">",
                ["get", "bounding_box_diagonal"],
                stepExpr as any
            ];
        }

        const startPoints: GeoJSON.FeatureCollection = {
            type: "FeatureCollection",
            features: geojson.features.map((f, i) => ({
                type: "Feature",
                properties: {
                    ...f.properties,
                    id: i
                },
                geometry: {
                    type: "Point",
                    coordinates: (f.geometry as any).coordinates[0]
                }
            }))
        };
        this.spec = {
            version: 8,
            name: "preview",
            glyphs: "https://tiles.openfreemap.org/fonts/{fontstack}/{range}.pbf",
            sources: {
                "preview": {
                    type: "geojson",
                    data: geojson,
                },
                "preview-start-points": {
                    type: "geojson",
                    data: startPoints,
                }
            },
            layers: [
                {
                    id: "preview",
                    type: "line",
                    source: "preview",
                    minzoom: minZoom,
                    filter: lineFilter,
                    paint: {
                        "line-color": ["get", "color"],
                        "line-width": 5,
                    },
                },
                {
                    id: "preview-start-points",
                    type: "circle",
                    source: "preview-start-points",
                    minzoom: minZoom,
                    paint: {
                        "circle-color": "#242734",
                        "circle-radius": 6,
                        "circle-stroke-width": 2,
                        "circle-stroke-color": "#fff",
                    },
                },
                {
                    id: "preview-direction-carets",
                    type: "symbol",
                    source: "preview",
                    minzoom: minZoom,
                    layout: {
                        "symbol-placement": "line",
                        "symbol-spacing": [
                            "interpolate",
                            ["exponential", 1.5],
                            ["zoom"],
                            0,
                            80,
                            18,
                            200,
                        ],
                        "icon-image": "direction-caret",
                        "icon-size": [
                            "interpolate",
                            ["exponential", 1.5],
                            ["zoom"],
                            0,
                            0.5,
                            18,
                            0.8,
                        ],
                    },
                }
            ]

        };
    }
}