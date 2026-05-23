import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail-merge/perfect-track:
 *   post:
 *     summary: Generate a perfect track from multiple trail IDs
 *     description: Returns a GPX string representing the averaged "perfect" track.
 *     tags:
 *       - Trail Merge
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               trailIds:
 *                 type: array
 *                 items:
 *                   type: string
 *     responses:
 *       200:
 *         description: Perfect track GPX
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 gpx:
 *                   type: string
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const response = await event.locals.pb.send("/trail-merge/perfect-track", {
            method: "POST",
            body,
            fetch: event.fetch,
        });

        return json(response);
    } catch (e: any) {
        return handleError(e);
    }
}
