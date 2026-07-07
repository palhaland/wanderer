import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail-merge/bulk:
 *   post:
 *     summary: Bulk merge trails
 *     description: Merges multiple source trails into a target trail, optionally generating a perfect track.
 *     tags:
 *       - Trail Merge
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               sourceTrailIds:
 *                 type: array
 *                 items:
 *                   type: string
 *               targetTrailId:
 *                 type: string
 *               settings:
 *                 type: object
 *               generatePerfectTrack:
 *                 type: boolean
 *     responses:
 *       200:
 *         description: Bulk merge acknowledged
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 acknowledged:
 *                   type: boolean
 *                 targetTrailId:
 *                   type: string
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       403:
 *         description: Forbidden
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const response = await event.locals.pb.send("/trail-merge/bulk", {
            method: "POST",
            body,
            fetch: event.fetch,
        });

        return json(response);
    } catch (e: any) {
        return handleError(e);
    }
}
