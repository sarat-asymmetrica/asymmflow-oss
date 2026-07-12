export interface ScreenLayout {
    id: string;
    title: string;
    type: string;
    components: Component[];
    theme: ThemeHints;
    grid_template: string;
    regime: VisualRegime;
}

export interface Component {
    id: string;
    type: string;
    data: any;
    grid_area: string;
    regime: number;
}

export interface ThemeHints {
    primary_color: string;
    accent_color: string;
    background_color: string;
}

export interface VisualRegime {
    name: string;
    primary_color: string;
    secondary_color: string;
    geometry: GeometryConfig;
    physics: PhysicsConfig;
    shader_uniforms: Record<string, number>;
}

export interface GeometryConfig {
    type: string;
    complexity: number;
    roughness: number;
    metalness: number;
}

export interface PhysicsConfig {
    flow_rate: number;
    turbulence: number;
    gravity: number;
    viscosity: number;
}
