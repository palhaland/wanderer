import type { MapMouseEvent, SourceSpecification, StyleSpecification } from "maplibre-gl";
import type { BaseLayer } from "./layers";
import * as M from "maplibre-gl";

export class ClusterLayer implements BaseLayer {

    spec: StyleSpecification;

    private map: M.Map;


    listeners: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }> = {
        "clusters": {
            onMouseUp: this.zoomOnCluster.bind(this),
            onEnter: () => this.map!.getCanvas().style.cursor = "pointer",
            onLeave: () => this.map!.getCanvas().style.cursor = ""
        },
        "unclustered-point": {
            onMouseUp: this.zoomOnUnclusteredPoint.bind(this),
            onEnter: () => this.map!.getCanvas().style.cursor = "pointer",
            onLeave: () => this.map!.getCanvas().style.cursor = ""
        }
    };

    constructor(map: M.Map, geojson: GeoJSON.FeatureCollection, maxZoom: number = 10, tiers?: { thresholds: number[], limits: number[] }, listeners?: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }>) {
        this.map = map;
        this.listeners = {
            "clusters": { ...this.listeners["clusters"], ...listeners?.["clusters"] },
            "unclustered-point": { ...this.listeners["unclustered-point"], ...listeners?.["unclustered-point"] }
        }

        const thresholds = tiers?.thresholds || [8, 10, 12];
        const limits = tiers?.limits || [25000, 10000, 5000];

        this.spec = {
            version: 8,
            name: "clusters",
            glyphs: "https://tiles.openfreemap.org/fonts/{fontstack}/{range}.pbf",
            sources: {
                "cluster-trails": {
                    type: "geojson",
                    data: geojson,
                    cluster: true,
                    clusterRadius: 50,
                    clusterMaxZoom: maxZoom,
                }
            },
            layers: [
                {
                    id: "clusters",
                    type: "circle",
                    source: "cluster-trails",
                    filter: ["has", "point_count"],
                    maxzoom: maxZoom + 1, // Allow clusters to stay a bit longer if needed
                    paint: {
                        "circle-color": "#242734",
                        "circle-radius": [
                            "step",
                            ["get", "point_count"],
                            10,
                            10,
                            15,
                            20,
                            20,
                            50,
                            25,
                            100,
                            30,
                            200,
                            35,
                        ],
                        "circle-stroke-width": 3,
                        "circle-stroke-color": "#fff",
                    },
                },
                {
                    id: "cluster-count",
                    type: "symbol",
                    source: "cluster-trails",
                    filter: ["has", "point_count"],
                    maxzoom: maxZoom + 1,
                    paint: {
                        "text-color": "#fff",
                    },
                    layout: {
                        "text-field": "{point_count_abbreviated}",
                        "text-font": ["Noto Sans Regular"],
                        "text-size": 12,
                    },
                },
                {
                    id: "unclustered-point",
                    type: "circle",
                    source: "cluster-trails",
                    filter: [
                        "all",
                        ["!", ["has", "point_count"]],
                        [
                            "<",
                            ["get", "bounding_box_diagonal"],
                            [
                                "step",
                                ["zoom"],
                                limits[0],
                                thresholds[0], limits[1],
                                thresholds[1], limits[2],
                                thresholds[2], 0
                            ]
                        ]
                    ],
                    paint: {
                        "circle-color": "#242734",
                        "circle-radius": 7,
                        "circle-stroke-width": 2,
                        "circle-stroke-color": "#fff",
                    },
                }
            ]

        };
    }

    private async zoomOnCluster(e: MapMouseEvent) {
        const features = this.map.queryRenderedFeatures(e.point, {
            layers: ["clusters"],
        });
        const clusterId = features[0].properties.cluster_id;
        const zoom = await (
            this.map.getSource("cluster-trails") as M.GeoJSONSource
        ).getClusterExpansionZoom(clusterId);
        this.map.flyTo({
            center: (features[0].geometry as any).coordinates,
            zoom,
            maxDuration: 3000
        });
    }

    private zoomOnUnclusteredPoint(e: MapMouseEvent) {
        const coordinates = (e as any).features[0].geometry.coordinates.slice();

        this.map.flyTo({
            center: coordinates,
            zoom: 12,
            maxDuration: 3000
        });
    }
}