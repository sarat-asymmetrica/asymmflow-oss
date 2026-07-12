
export class Quaternion {
    w: number;
    x: number;
    y: number;
    z: number;

    constructor(w: number, x: number, y: number, z: number) {
        this.w = w;
        this.x = x;
        this.y = y;
        this.z = z;
    }

    norm(): number {
        return Math.sqrt(this.w * this.w + this.x * this.x + this.y * this.y + this.z * this.z);
    }

    normalize(): Quaternion {
        const n = this.norm();
        if (n < 1e-10) return new Quaternion(1, 0, 0, 0);
        return new Quaternion(this.w / n, this.x / n, this.y / n, this.z / n);
    }

    dot(q: Quaternion): number {
        return this.w * q.w + this.x * q.x + this.y * q.y + this.z * q.z;
    }

    multiply(q: Quaternion): Quaternion {
        return new Quaternion(
            this.w * q.w - this.x * q.x - this.y * q.y - this.z * q.z,
            this.w * q.x + this.x * q.w + this.y * q.z - this.z * q.y,
            this.w * q.y - this.x * q.z + this.y * q.w + this.z * q.x,
            this.w * q.z + this.x * q.y - this.y * q.x + this.z * q.w
        );
    }

    scale(s: number): Quaternion {
        return new Quaternion(this.w * s, this.x * s, this.y * s, this.z * s);
    }

    add(q: Quaternion): Quaternion {
        return new Quaternion(this.w + q.w, this.x + q.x, this.y + q.y, this.z + q.z);
    }

    static slerp(qa: Quaternion, qb: Quaternion, t: number): Quaternion {
        let dot = qa.dot(qb);
        let q1 = qb;

        if (dot < 0) {
            dot = -dot;
            q1 = qb.scale(-1);
        }

        if (dot > 0.9995) {
            return qa.add(q1.add(qa.scale(-1)).scale(t)).normalize();
        }

        const theta = Math.acos(dot);
        const sinTheta = Math.sin(theta);

        const w0 = Math.sin((1 - t) * theta) / sinTheta;
        const w1 = Math.sin(t * theta) / sinTheta;

        return qa.scale(w0).add(q1.scale(w1));
    }

    toCSSColor(): string {
       // Map quaternion to RGB? Or just use it for interpolation of theme values?
       // For theme interpolation, we typically interpolate 4 values (e.g. R, G, B, A or H, S, L, A)
       // Let's assume the quaternion represents a point in a 4D color space or state space.
       // For this task, we mainly need SLERP for the theme transition.
       return `rgba(${Math.abs(this.x)*255}, ${Math.abs(this.y)*255}, ${Math.abs(this.z)*255}, ${Math.abs(this.w)})`;
    }
}
