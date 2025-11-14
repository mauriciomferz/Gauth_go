/**
 * PoA Visualization Module
 * Star-inspired 3D visualization for Power of Attorney relationships
 * with particle systems, mouse zoom/navigation, and brightness/color mapping
 */

import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';

/**
 * PoA Graph Visualizer
 * Renders PoA relationship graphs as stars in 3D space with particle systems
 */
export class PoAGraphVisualizer {
    constructor(container) {
        this.container = container;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        this.nodes = new Map();
        this.edges = [];
        this.particleSystems = new Map();
        this.animationId = null;
        this.starField = null;
        
        this.init();
    }
    
    init() {
        // Scene with Star Wars deep space background
        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x000000); // Pure black space
        this.scene.fog = new THREE.FogExp2(0x000814, 0.008); // Very subtle blue fog
        
        // Camera
        const aspect = this.container.clientWidth / this.container.clientHeight;
        this.camera = new THREE.PerspectiveCamera(75, aspect, 0.1, 1000);
        this.camera.position.set(15, 15, 15);
        
        // Renderer
        this.renderer = new THREE.WebGLRenderer({ 
            antialias: true, 
            alpha: true,
            powerPreference: 'high-performance'
        });
        this.renderer.setSize(this.container.clientWidth, this.container.clientHeight);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        this.container.appendChild(this.renderer.domElement);
        
        // Enhanced Controls with zoom and navigation
        this.controls = new OrbitControls(this.camera, this.renderer.domElement);
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.08;
        this.controls.screenSpacePanning = false;
        this.controls.minDistance = 2;
        this.controls.maxDistance = 100;
        this.controls.maxPolarAngle = Math.PI;
        
        // Smooth zoom controls
        this.controls.zoomSpeed = 1.2;
        this.controls.rotateSpeed = 0.5;
        this.controls.panSpeed = 0.8;
        
        // Enable all navigation
        this.controls.enableZoom = true;
        this.controls.enableRotate = true;
        this.controls.enablePan = true;
        
        // Lighting for star-like environment
        this.setupStarLighting();
        
        // Create background star field
        this.createStarField();
        
        // Handle resize
        window.addEventListener('resize', () => this.onResize());
        
        // Add mouse interaction
        this.setupMouseInteraction();
        
        // Start animation loop
        this.animate();
    }
    
    setupStarLighting() {
        // Ambient light with cool Star Wars space glow
        const ambientLight = new THREE.AmbientLight(0x1a1a2e, 0.4);
        this.scene.add(ambientLight);
        
        // Directional light (distant sun/star)
        const sunLight = new THREE.DirectionalLight(0xccddff, 0.6);
        sunLight.position.set(50, 50, 50);
        this.scene.add(sunLight);
        
        // Point lights for Star Wars atmosphere (blue/white)
        const starLight1 = new THREE.PointLight(0x6699ff, 1.8, 100);
        starLight1.position.set(20, 20, 20);
        this.scene.add(starLight1);
        
        const starLight2 = new THREE.PointLight(0xffffff, 1.2, 100);
        starLight2.position.set(-20, 20, -20);
        this.scene.add(starLight2);
        
        const starLight3 = new THREE.PointLight(0x8899ff, 1.0, 100);
        starLight3.position.set(0, -20, 20);
        this.scene.add(starLight3);
    }
    
    /**
     * Create Star Wars deep space starfield background
     */
    createStarField() {
        const starGeometry = new THREE.BufferGeometry();
        const particleCount = 5000; // Dense starfield like original Star Wars
        const positions = new Float32Array(particleCount * 3);
        const colors = new Float32Array(particleCount * 3);
        const sizes = new Float32Array(particleCount);
        const velocities = [];
        
        for (let i = 0; i < particleCount; i++) {
            const i3 = i * 3;
            
            // Random spherical distribution (distant stars)
            const theta = Math.random() * Math.PI * 2;
            const phi = Math.acos((Math.random() * 2) - 1);
            const radius = 50 + Math.random() * 40;
            
            positions[i3] = radius * Math.sin(phi) * Math.cos(theta);
            positions[i3 + 1] = radius * Math.sin(phi) * Math.sin(theta);
            positions[i3 + 2] = radius * Math.cos(phi);
            
            // Star Wars color palette: mostly white/blue stars with some yellow
            const colorType = Math.random();
            if (colorType < 0.7) {
                // White stars (most common)
                const whiteness = 0.8 + Math.random() * 0.2;
                colors[i3] = whiteness;
                colors[i3 + 1] = whiteness;
                colors[i3 + 2] = whiteness;
            } else if (colorType < 0.9) {
                // Blue-white stars
                colors[i3] = 0.7 + Math.random() * 0.2;
                colors[i3 + 1] = 0.8 + Math.random() * 0.2;
                colors[i3 + 2] = 0.9 + Math.random() * 0.1;
            } else {
                // Yellow stars (distant suns)
                colors[i3] = 0.9 + Math.random() * 0.1;
                colors[i3 + 1] = 0.8 + Math.random() * 0.2;
                colors[i3 + 2] = 0.5 + Math.random() * 0.3;
            }
            
            // Varied star sizes (some bright, some dim)
            const brightness = Math.random();
            if (brightness > 0.95) {
                sizes[i] = 0.3 + Math.random() * 0.2; // Bright stars
            } else if (brightness > 0.8) {
                sizes[i] = 0.15 + Math.random() * 0.1; // Medium stars
            } else {
                sizes[i] = 0.05 + Math.random() * 0.05; // Distant stars
            }
            
            // Very slow rotation (space is vast and still)
            velocities.push({
                x: (Math.random() - 0.5) * 0.002,
                y: (Math.random() - 0.5) * 0.002,
                z: (Math.random() - 0.5) * 0.002
            });
        }
        
        starGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
        starGeometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
        starGeometry.setAttribute('size', new THREE.BufferAttribute(sizes, 1));
        
        const starMaterial = new THREE.PointsMaterial({
            size: 0.2,
            vertexColors: true,
            transparent: true,
            opacity: 0.9,
            sizeAttenuation: true,
            blending: THREE.AdditiveBlending
        });
        
        this.starField = new THREE.Points(starGeometry, starMaterial);
        this.starField.userData.velocities = velocities;
        this.scene.add(this.starField);
    }
    
    /**
     * Setup mouse interaction for node selection
     */
    setupMouseInteraction() {
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        
        this.renderer.domElement.addEventListener('mousemove', (event) => {
            const rect = this.renderer.domElement.getBoundingClientRect();
            this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
            this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;
        });
        
        this.renderer.domElement.addEventListener('click', (event) => {
            this.raycaster.setFromCamera(this.mouse, this.camera);
            const intersects = this.raycaster.intersectObjects(Array.from(this.nodes.values()));
            
            if (intersects.length > 0) {
                const node = intersects[0].object;
                this.highlightNode(node);
            }
        });
    }
    
    /**
     * Load and render a PoA graph with star coordinates
     */
    async loadGraph(graphData) {
        // Clear existing visualization
        this.clearGraph();
        
        // Parse nodes and convert to 3D star coordinates
        const nodes = this.parseNodesToStarCoordinates(graphData.nodes);
        
        // Create star nodes
        for (const node of nodes) {
            this.createStarNode(node);
        }
        
        // Create edges
        for (const edge of graphData.edges) {
            this.createEdge(edge);
        }
    }
    
    /**
     * Parse node data into 3D star coordinates
     * Distributes nodes in a spherical/galactic pattern
     */
    parseNodesToStarCoordinates(nodes) {
        const parsedNodes = [];
        const radius = 10;
        const nodeCount = nodes.length;
        
        nodes.forEach((node, index) => {
            // Use provided position or calculate spherical coordinates
            let position;
            
            if (node.position && node.position.x !== undefined) {
                position = node.position;
            } else {
                // Golden spiral distribution for even spacing
                const phi = Math.acos(1 - 2 * (index + 0.5) / nodeCount);
                const theta = Math.PI * (1 + Math.sqrt(5)) * index;
                
                // Add some randomness for natural star field look
                const r = radius * (0.8 + Math.random() * 0.4);
                
                position = {
                    x: r * Math.sin(phi) * Math.cos(theta),
                    y: r * Math.sin(phi) * Math.sin(theta),
                    z: r * Math.cos(phi)
                };
            }
            
            parsedNodes.push({
                ...node,
                position: position,
                brightness: this.calculateBrightness(node),
                starColor: this.getStarColor(node)
            });
        });
        
        return parsedNodes;
    }
    
    /**
     * Calculate star brightness based on node properties
     */
    calculateBrightness(node) {
        let brightness = 1.0;
        
        // Status affects brightness
        switch (node.status) {
            case 'active':
                brightness = 1.5;
                break;
            case 'pending':
                brightness = 1.0;
                break;
            case 'revoked':
                brightness = 0.5;
                break;
            case 'expired':
                brightness = 0.3;
                break;
        }
        
        // Type affects brightness
        if (node.type === 'principal') brightness *= 1.3;
        if (node.type === 'resource') brightness *= 0.8;
        
        return brightness;
    }
    
    /**
     * Get star color with temperature mapping
     */
    getStarColor(node) {
        // Color temperature mapping (like real stars)
        const colors = {
            active: { r: 0.4, g: 0.7, b: 1.0 },      // Blue (hot)
            pending: { r: 1.0, g: 0.85, b: 0.4 },    // Yellow
            revoked: { r: 1.0, g: 0.2, b: 0.2 },     // Red (cool)
            expired: { r: 0.6, g: 0.6, b: 0.6 }      // White-gray
        };
        
        return colors[node.status] || { r: 0.9, g: 0.9, b: 1.0 };
    }
    
    /**
     * Get NASA-style celestial texture based on node type and status
     */
    getCelestialTexture(nodeData) {
        const canvas = document.createElement('canvas');
        canvas.width = 512;
        canvas.height = 512;
        const ctx = canvas.getContext('2d');
        
        // Create procedural NASA-style textures
        switch (nodeData.type) {
            case 'principal':
                return this.createStarTexture(ctx, nodeData.status);
            case 'client':
                return this.createPlanetTexture(ctx, nodeData.status);
            case 'resource':
                return this.createCometTexture(ctx, nodeData.status);
            default:
                return this.createNebulaTexture(ctx, nodeData.status);
        }
    }
    
    /**
     * Create NASA-style star texture (Sun-like)
     */
    createStarTexture(ctx, status) {
        const canvas = ctx.canvas;
        const centerX = canvas.width / 2;
        const centerY = canvas.height / 2;
        const radius = canvas.width / 2;
        
        // Star color based on status
        const colors = {
            active: ['#FFD700', '#FFA500', '#FF6B00'],      // Yellow/Orange (Sun-like)
            pending: ['#FFFF00', '#FFD700', '#FFA500'],     // Yellow
            revoked: ['#FF4500', '#DC143C', '#8B0000'],     // Red giant
            expired: ['#E0E0E0', '#B0B0B0', '#808080']      // White dwarf
        };
        
        const starColors = colors[status] || colors.active;
        
        // Create radial gradient for star surface
        const gradient = ctx.createRadialGradient(centerX, centerY, 0, centerX, centerY, radius);
        gradient.addColorStop(0, starColors[0]);
        gradient.addColorStop(0.4, starColors[1]);
        gradient.addColorStop(0.8, starColors[2]);
        gradient.addColorStop(1, 'rgba(0,0,0,0)');
        
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        
        // Add surface details (solar flares/spots)
        for (let i = 0; i < 30; i++) {
            const angle = Math.random() * Math.PI * 2;
            const dist = Math.random() * radius * 0.7;
            const x = centerX + Math.cos(angle) * dist;
            const y = centerY + Math.sin(angle) * dist;
            const size = Math.random() * 20 + 10;
            
            const spotGradient = ctx.createRadialGradient(x, y, 0, x, y, size);
            spotGradient.addColorStop(0, 'rgba(255, 200, 0, 0.3)');
            spotGradient.addColorStop(1, 'rgba(255, 200, 0, 0)');
            ctx.fillStyle = spotGradient;
            ctx.fillRect(x - size, y - size, size * 2, size * 2);
        }
        
        return new THREE.CanvasTexture(canvas);
    }
    
    /**
     * Create NASA-style planet texture (Earth/Mars-like)
     */
    createPlanetTexture(ctx, status) {
        const canvas = ctx.canvas;
        const centerX = canvas.width / 2;
        const centerY = canvas.height / 2;
        const radius = canvas.width / 2;
        
        // Planet colors based on status
        const planetTypes = {
            active: { base: '#4169E1', secondary: '#228B22', tertiary: '#87CEEB' },  // Earth-like
            pending: { base: '#DAA520', secondary: '#CD853F', tertiary: '#F4A460' }, // Mars-like
            revoked: { base: '#696969', secondary: '#404040', tertiary: '#A9A9A9' }, // Dead planet
            expired: { base: '#2F4F4F', secondary: '#1C1C1C', tertiary: '#4B4B4B' }  // Dark planet
        };
        
        const colors = planetTypes[status] || planetTypes.active;
        
        // Base planet surface
        const gradient = ctx.createRadialGradient(centerX - 50, centerY - 50, 0, centerX, centerY, radius);
        gradient.addColorStop(0, colors.tertiary);
        gradient.addColorStop(0.5, colors.base);
        gradient.addColorStop(1, colors.secondary);
        
        ctx.fillStyle = gradient;
        ctx.beginPath();
        ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
        ctx.fill();
        
        // Add continents/terrain features
        for (let i = 0; i < 15; i++) {
            const angle = Math.random() * Math.PI * 2;
            const dist = Math.random() * radius * 0.8;
            const x = centerX + Math.cos(angle) * dist;
            const y = centerY + Math.sin(angle) * dist;
            const size = Math.random() * 60 + 30;
            
            ctx.fillStyle = colors.secondary;
            ctx.globalAlpha = 0.6;
            ctx.beginPath();
            ctx.arc(x, y, size, 0, Math.PI * 2);
            ctx.fill();
        }
        
        ctx.globalAlpha = 1.0;
        
        // Add atmosphere glow
        const atmoGradient = ctx.createRadialGradient(centerX, centerY, radius * 0.9, centerX, centerY, radius * 1.1);
        atmoGradient.addColorStop(0, 'rgba(135, 206, 250, 0)');
        atmoGradient.addColorStop(1, 'rgba(135, 206, 250, 0.3)');
        ctx.fillStyle = atmoGradient;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        
        return new THREE.CanvasTexture(canvas);
    }
    
    /**
     * Create NASA-style comet texture
     */
    createCometTexture(ctx, status) {
        const canvas = ctx.canvas;
        const centerX = canvas.width / 3;
        const centerY = canvas.height / 2;
        
        // Comet core colors
        const coreColors = {
            active: ['#FFFFFF', '#E0E0E0', '#B0B0B0'],
            pending: ['#FFE4B5', '#FFDAB9', '#D2B48C'],
            revoked: ['#696969', '#505050', '#303030'],
            expired: ['#808080', '#606060', '#404040']
        };
        
        const colors = coreColors[status] || coreColors.active;
        
        // Draw comet nucleus
        const coreGradient = ctx.createRadialGradient(centerX, centerY, 0, centerX, centerY, 40);
        coreGradient.addColorStop(0, colors[0]);
        coreGradient.addColorStop(0.6, colors[1]);
        coreGradient.addColorStop(1, colors[2]);
        
        ctx.fillStyle = coreGradient;
        ctx.beginPath();
        ctx.arc(centerX, centerY, 40, 0, Math.PI * 2);
        ctx.fill();
        
        // Draw comet tail
        for (let i = 0; i < 20; i++) {
            const tailX = centerX + (i * 20);
            const tailY = centerY + (Math.random() - 0.5) * 50;
            const tailSize = 30 + i * 5;
            
            const tailGradient = ctx.createRadialGradient(tailX, tailY, 0, tailX, tailY, tailSize);
            tailGradient.addColorStop(0, `rgba(255, 255, 255, ${0.3 - i * 0.015})`);
            tailGradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
            
            ctx.fillStyle = tailGradient;
            ctx.fillRect(tailX - tailSize, tailY - tailSize, tailSize * 2, tailSize * 2);
        }
        
        return new THREE.CanvasTexture(canvas);
    }
    
    /**
     * Create NASA-style nebula texture
     */
    createNebulaTexture(ctx, status) {
        const canvas = ctx.canvas;
        
        // Nebula colors based on status
        const nebulaColors = {
            active: ['#FF1493', '#9370DB', '#4169E1'],      // Pink/Purple nebula
            pending: ['#FFD700', '#FF8C00', '#FF4500'],     // Orange nebula
            revoked: ['#DC143C', '#8B0000', '#4B0000'],     // Red nebula
            expired: ['#708090', '#2F4F4F', '#191970']      // Dark nebula
        };
        
        const colors = nebulaColors[status] || nebulaColors.active;
        
        // Create nebula clouds
        for (let i = 0; i < 50; i++) {
            const x = Math.random() * canvas.width;
            const y = Math.random() * canvas.height;
            const size = Math.random() * 100 + 50;
            const colorIndex = Math.floor(Math.random() * colors.length);
            
            const cloudGradient = ctx.createRadialGradient(x, y, 0, x, y, size);
            cloudGradient.addColorStop(0, colors[colorIndex] + '80');
            cloudGradient.addColorStop(1, 'rgba(0, 0, 0, 0)');
            
            ctx.fillStyle = cloudGradient;
            ctx.fillRect(x - size, y - size, size * 2, size * 2);
        }
        
        // Add bright spots (stars within nebula)
        for (let i = 0; i < 30; i++) {
            const x = Math.random() * canvas.width;
            const y = Math.random() * canvas.height;
            const size = Math.random() * 3 + 1;
            
            ctx.fillStyle = '#FFFFFF';
            ctx.beginPath();
            ctx.arc(x, y, size, 0, Math.PI * 2);
            ctx.fill();
        }
        
        return new THREE.CanvasTexture(canvas);
    }
    
    /**
     * Create a celestial body node with NASA-style texture
     */
    createStarNode(nodeData) {
        const position = nodeData.position;
        const brightness = nodeData.brightness || 1.0;
        const starColor = nodeData.starColor;
        
        // Determine size based on type
        let size = 0.4 * brightness;
        if (nodeData.type === 'principal') size *= 1.3;
        if (nodeData.type === 'resource') size *= 0.8;
        
        // Create geometry
        const coreGeometry = new THREE.SphereGeometry(size, 32, 32);
        
        // Get NASA-style texture
        const texture = this.getCelestialTexture(nodeData);
        const color = new THREE.Color(starColor.r, starColor.g, starColor.b);
        
        // Create material with texture
        const coreMaterial = new THREE.MeshPhongMaterial({
            map: texture,
            emissive: color,
            emissiveIntensity: brightness * 0.5,
            shininess: 50,
            transparent: true,
            opacity: 0.95
        });
        
        const core = new THREE.Mesh(coreGeometry, coreMaterial);
        core.position.set(position.x, position.y, position.z);
        core.userData = nodeData;
        
        // Add point light for celestial glow
        const pointLight = new THREE.PointLight(color, brightness * 2, 10);
        pointLight.position.set(0, 0, 0);
        core.add(pointLight);
        
        // Add atmospheric glow layers
        for (let i = 0; i < 2; i++) {
            const glowSize = size + (i * 0.15);
            const glowGeometry = new THREE.SphereGeometry(glowSize, 16, 16);
            const glowMaterial = new THREE.MeshBasicMaterial({
                color: color,
                transparent: true,
                opacity: 0.1 / (i + 1),
                side: THREE.BackSide,
                blending: THREE.AdditiveBlending
            });
            const glow = new THREE.Mesh(glowGeometry, glowMaterial);
            core.add(glow);
        }
        
        // Add to scene
        this.scene.add(core);
        this.nodes.set(nodeData.id, core);
        
        // Create particle system for the celestial body
        this.createParticleSystem(nodeData, position, color, brightness);
        
        // Add label
        this.createLabel(nodeData.label, position);
        
        return core;
    }
    
    /**
     * Create Star Wars energy field particle system around node
     */
    createParticleSystem(nodeData, position, color, brightness) {
        const particleCount = Math.floor(50 * brightness);
        const geometry = new THREE.BufferGeometry();
        const positions = new Float32Array(particleCount * 3);
        const colors = new Float32Array(particleCount * 3);
        const velocities = [];
        
        for (let i = 0; i < particleCount; i++) {
            const i3 = i * 3;
            
            // Random spherical distribution around node
            const radius = 0.5 + Math.random() * 1.5;
            const theta = Math.random() * Math.PI * 2;
            const phi = Math.acos((Math.random() * 2) - 1);
            
            positions[i3] = radius * Math.sin(phi) * Math.cos(theta);
            positions[i3 + 1] = radius * Math.sin(phi) * Math.sin(theta);
            positions[i3 + 2] = radius * Math.cos(phi);
            
            // Star Wars energy colors: blue, white, cyan (like lightsabers and ship lights)
            const colorType = Math.random();
            if (colorType < 0.5) {
                // Electric blue (classic Star Wars blue)
                colors[i3] = 0.3 + Math.random() * 0.2;
                colors[i3 + 1] = 0.6 + Math.random() * 0.3;
                colors[i3 + 2] = 0.9 + Math.random() * 0.1;
            } else if (colorType < 0.8) {
                // Bright white (energy cores)
                const intensity = 0.8 + Math.random() * 0.2;
                colors[i3] = intensity;
                colors[i3 + 1] = intensity;
                colors[i3 + 2] = intensity;
            } else {
                // Cyan/turquoise (control panels)
                colors[i3] = 0.2 + Math.random() * 0.3;
                colors[i3 + 1] = 0.7 + Math.random() * 0.2;
                colors[i3 + 2] = 0.8 + Math.random() * 0.2;
            }
            
            // Store velocity for orbital motion
            velocities.push({
                x: (Math.random() - 0.5) * 0.012,
                y: (Math.random() - 0.5) * 0.012,
                z: (Math.random() - 0.5) * 0.012
            });
        }
        
        geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
        geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
        
        const material = new THREE.PointsMaterial({
            vertexColors: true,
            size: 0.008, // Minimal energy particles (smallest possible)
            transparent: true,
            opacity: 0.8,
            blending: THREE.AdditiveBlending,
            sizeAttenuation: true
        });
        
        const particles = new THREE.Points(geometry, material);
        particles.position.set(position.x, position.y, position.z);
        particles.userData = { velocities: velocities, nodeId: nodeData.id };
        
        this.scene.add(particles);
        this.particleSystems.set(nodeData.id, particles);
    }
    
    /**
     * Create edge between nodes
     */
    createEdge(edgeData) {
        const sourceNode = this.nodes.get(edgeData.source);
        const targetNode = this.nodes.get(edgeData.target);
        
        if (!sourceNode || !targetNode) return;
        
        const points = [];
        points.push(sourceNode.position);
        points.push(targetNode.position);
        
        // Curved line for visual interest
        const curve = new THREE.QuadraticBezierCurve3(
            sourceNode.position,
            new THREE.Vector3(
                (sourceNode.position.x + targetNode.position.x) / 2,
                Math.max(sourceNode.position.y, targetNode.position.y) + 1,
                (sourceNode.position.z + targetNode.position.z) / 2
            ),
            targetNode.position
        );
        
        const curvePoints = curve.getPoints(50);
        const geometry = new THREE.BufferGeometry().setFromPoints(curvePoints);
        
        // Edge color based on type
        const color = this.getEdgeColor(edgeData.type);
        const material = new THREE.LineBasicMaterial({
            color: color,
            transparent: true,
            opacity: edgeData.strength || 0.6,
            linewidth: 2
        });
        
        const line = new THREE.Line(geometry, material);
        line.userData = edgeData;
        this.scene.add(line);
        this.edges.push(line);
        
        // Add arrow at target
        this.createArrow(targetNode.position, sourceNode.position, color);
    }
    
    /**
     * Create directional arrow
     */
    createArrow(target, source, color) {
        const direction = new THREE.Vector3()
            .subVectors(target, source)
            .normalize();
        
        const arrowHelper = new THREE.ArrowHelper(
            direction,
            target.clone().sub(direction.multiplyScalar(0.6)),
            0.5,
            color,
            0.2,
            0.15
        );
        
        this.scene.add(arrowHelper);
    }
    
    /**
     * Create text label (sprite-based)
     */
    createLabel(text, position) {
        const canvas = document.createElement('canvas');
        const context = canvas.getContext('2d');
        canvas.width = 256;
        canvas.height = 64;
        
        context.fillStyle = '#ffffff';
        context.font = 'Bold 24px Arial';
        context.textAlign = 'center';
        context.fillText(text, 128, 32);
        
        const texture = new THREE.CanvasTexture(canvas);
        const spriteMaterial = new THREE.SpriteMaterial({ 
            map: texture,
            transparent: true,
            opacity: 0.8
        });
        
        const sprite = new THREE.Sprite(spriteMaterial);
        sprite.position.set(position.x, position.y + 1, position.z);
        sprite.scale.set(2, 0.5, 1);
        
        this.scene.add(sprite);
    }
    
    /**
     * Get color based on node status
     */
    getStatusColor(status) {
        const colors = {
            active: 0x10b981,
            pending: 0xf59e0b,
            revoked: 0xef4444,
            expired: 0x6b7280
        };
        return colors[status] || 0x667eea;
    }
    
    /**
     * Get color based on edge type
     */
    getEdgeColor(type) {
        const colors = {
            authorizes: 0x667eea,
            delegates: 0xf59e0b,
            requests: 0x10b981,
            validates: 0x3b82f6
        };
        return colors[type] || 0x8b5cf6;
    }
    
    /**
     * Highlight a selected node
     */
    highlightNode(node) {
        // Reset all nodes
        this.nodes.forEach(n => {
            if (n.material) {
                n.material.emissiveIntensity = 0.3;
            }
        });
        
        // Highlight selected node
        if (node.material) {
            node.material.emissiveIntensity = 1.5;
        }
        
        // Show node info
        console.log('Selected node:', node.userData);
    }
    
    /**
     * Clear the graph and particle systems
     */
    clearGraph() {
        // Remove nodes
        this.nodes.forEach(node => {
            this.scene.remove(node);
        });
        this.nodes.clear();
        
        // Remove particle systems
        this.particleSystems.forEach(particles => {
            this.scene.remove(particles);
            if (particles.geometry) particles.geometry.dispose();
            if (particles.material) particles.material.dispose();
        });
        this.particleSystems.clear();
        
        // Remove edges
        this.edges.forEach(edge => {
            this.scene.remove(edge);
            if (edge.geometry) edge.geometry.dispose();
            if (edge.material) edge.material.dispose();
        });
        this.edges = [];
        
        // Clear other objects except star field and lights
        const objectsToRemove = [];
        this.scene.traverse(obj => {
            if (obj.type === 'Sprite' || obj.type === 'ArrowHelper') {
                objectsToRemove.push(obj);
            }
        });
        objectsToRemove.forEach(obj => this.scene.remove(obj));
    }
    
    /**
     * Animation loop with particle system updates
     */
    animate() {
        this.animationId = requestAnimationFrame(() => this.animate());
        
        const time = Date.now() * 0.001;
        
        // Animate star nodes
        this.nodes.forEach(node => {
            // Gentle rotation
            node.rotation.y += 0.001;
            
            // Pulse effect for pending nodes
            if (node.userData.status === 'pending') {
                const scale = 1 + Math.sin(time * 2) * 0.1;
                node.scale.set(scale, scale, scale);
            }
            
            // Twinkle effect for active nodes
            if (node.userData.status === 'active' && node.children.length > 0) {
                const light = node.children.find(child => child.type === 'PointLight');
                if (light) {
                    light.intensity = node.userData.brightness * 2 * (0.8 + Math.sin(time * 3 + node.position.x) * 0.2);
                }
            }
        });
        
        // Animate particle systems
        this.particleSystems.forEach(particles => {
            const positions = particles.geometry.attributes.position.array;
            const velocities = particles.userData.velocities;
            
            for (let i = 0; i < velocities.length; i++) {
                const i3 = i * 3;
                
                // Update position based on velocity
                positions[i3] += velocities[i].x;
                positions[i3 + 1] += velocities[i].y;
                positions[i3 + 2] += velocities[i].z;
                
                // Keep particles within bounds (orbital motion)
                const distance = Math.sqrt(
                    positions[i3] * positions[i3] +
                    positions[i3 + 1] * positions[i3 + 1] +
                    positions[i3 + 2] * positions[i3 + 2]
                );
                
                if (distance > 2.0) {
                    positions[i3] *= 0.95;
                    positions[i3 + 1] *= 0.95;
                    positions[i3 + 2] *= 0.95;
                } else if (distance < 0.5) {
                    positions[i3] *= 1.05;
                    positions[i3 + 1] *= 1.05;
                    positions[i3 + 2] *= 1.05;
                }
            }
            
            particles.geometry.attributes.position.needsUpdate = true;
            
            // Rotate particle system
            particles.rotation.y += 0.001;
        });
        
        // Animate Star Wars starfield (slow parallax rotation)
        if (this.starField && this.starField.userData.velocities) {
            const positions = this.starField.geometry.attributes.position.array;
            const velocities = this.starField.userData.velocities;
            
            for (let i = 0; i < velocities.length; i++) {
                const i3 = i * 3;
                
                // Very slow drift (stars are distant and static)
                positions[i3] += velocities[i].x;
                positions[i3 + 1] += velocities[i].y;
                positions[i3 + 2] += velocities[i].z;
                
                // Keep stars in spherical bounds (50-90 units from center)
                const distSquared = positions[i3]**2 + positions[i3+1]**2 + positions[i3+2]**2;
                if (distSquared > 8100) { // 90^2
                    // Pull back toward inner sphere
                    const scale = 0.99;
                    positions[i3] *= scale;
                    positions[i3 + 1] *= scale;
                    positions[i3 + 2] *= scale;
                } else if (distSquared < 2500) { // 50^2
                    // Push out toward outer sphere
                    const scale = 1.01;
                    positions[i3] *= scale;
                    positions[i3 + 1] *= scale;
                    positions[i3 + 2] *= scale;
                }
            }
            
            this.starField.geometry.attributes.position.needsUpdate = true;
            
            // Very slow rotation (camera/ship moving through space)
            this.starField.rotation.y += 0.00008;
            this.starField.rotation.x += 0.00003;
        }
        
        this.controls.update();
        this.renderer.render(this.scene, this.camera);
    }
    
    /**
     * Handle window resize
     */
    onResize() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;
        
        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(width, height);
    }
    
    /**
     * Dispose resources including particles
     */
    dispose() {
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
        }
        
        // Clear all graph elements
        this.clearGraph();
        
        // Dispose star field
        if (this.starField) {
            this.scene.remove(this.starField);
            if (this.starField.geometry) this.starField.geometry.dispose();
            if (this.starField.material) this.starField.material.dispose();
        }
        
        // Dispose renderer and controls
        this.renderer.dispose();
        this.controls.dispose();
        
        if (this.renderer.domElement.parentNode) {
            this.renderer.domElement.parentNode.removeChild(this.renderer.domElement);
        }
    }
}

/**
 * Protocol Step Visualizer
 * 3D layered visualization of protocol steps
 */
export class ProtocolStepVisualizer {
    constructor(container) {
        this.container = container;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        this.layers = [];
        this.components = new Map();
        this.connections = [];
        this.animationId = null;
        
        this.init();
    }
    
    init() {
        // Scene
        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x000000); // Pure black space
        this.scene.fog = new THREE.FogExp2(0x000814, 0.008); // Very subtle blue fog
        
        // Camera
        const aspect = this.container.clientWidth / this.container.clientHeight;
        this.camera = new THREE.PerspectiveCamera(50, aspect, 0.1, 1000);
        this.camera.position.set(8, 8, 8);
        
        // Renderer
        this.renderer = new THREE.WebGLRenderer({ 
            antialias: true,
            alpha: true
        });
        this.renderer.setSize(this.container.clientWidth, this.container.clientHeight);
        this.renderer.setPixelRatio(window.devicePixelRatio);
        this.container.appendChild(this.renderer.domElement);
        
        // Controls
        this.controls = new OrbitControls(this.camera, this.renderer.domElement);
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.05;
        this.controls.minDistance = 5;
        this.controls.maxDistance = 30;
        
        // Lighting
        this.setupLighting();
        
        // Handle resize
        window.addEventListener('resize', () => this.onResize());
        
        // Start animation
        this.animate();
    }
    
    setupLighting() {
        // Cool blue-gray ambient (Star Wars space atmosphere)
        const ambientLight = new THREE.AmbientLight(0x1a1a2e, 0.4);
        this.scene.add(ambientLight);
        
        // Blue-white distant sun
        const directionalLight = new THREE.DirectionalLight(0xccddff, 0.6);
        directionalLight.position.set(5, 10, 5);
        this.scene.add(directionalLight);
        
        // Blue and white Star Wars accent lights
        const accentLight1 = new THREE.PointLight(0x6699ff, 1.8, 100);
        accentLight1.position.set(10, 10, 10);
        this.scene.add(accentLight1);
        
        const accentLight2 = new THREE.PointLight(0xffffff, 1.2, 100);
        accentLight2.position.set(-10, 10, -10);
        this.scene.add(accentLight2);
        
        const accentLight3 = new THREE.PointLight(0x8899ff, 1.0, 100);
        accentLight3.position.set(0, -10, 0);
        this.scene.add(accentLight3);
    }
    
    /**
     * Load protocol step visualization
     */
    async loadProtocolStep(stepData) {
        this.clearVisualization();
        
        // Create layers
        for (const layer of stepData.layers) {
            this.createLayer(layer);
        }
        
        // Create connections
        for (const connection of stepData.connections) {
            this.createConnection(connection);
        }
        
        // Focus camera on the visualization
        this.focusOnVisualization();
    }
    
    /**
     * Create a visualization layer
     */
    createLayer(layerData) {
        const group = new THREE.Group();
        group.userData = layerData;
        
        // Create layer platform
        const platformGeometry = new THREE.CylinderGeometry(4, 4, 0.1, 32);
        const platformMaterial = new THREE.MeshPhongMaterial({
            color: parseInt(layerData.color.replace('#', '0x')),
            transparent: true,
            opacity: layerData.opacity * 0.3,
            emissive: parseInt(layerData.color.replace('#', '0x')),
            emissiveIntensity: 0.2
        });
        
        const platform = new THREE.Mesh(platformGeometry, platformMaterial);
        platform.position.y = layerData.level * 3;
        group.add(platform);
        
        // Create components on this layer
        for (const component of layerData.components) {
            const comp = this.createComponent(component, layerData.level);
            group.add(comp);
            this.components.set(component.id, comp);
        }
        
        this.scene.add(group);
        this.layers.push(group);
    }
    
    /**
     * Create a layer component
     */
    createComponent(componentData, level) {
        const group = new THREE.Group();
        group.userData = componentData;
        
        // Component mesh based on type
        let geometry;
        switch (componentData.type) {
            case 'actor':
                geometry = new THREE.ConeGeometry(0.4, 0.8, 4);
                break;
            case 'process':
                geometry = new THREE.BoxGeometry(0.6, 0.6, 0.6);
                break;
            case 'data':
                geometry = new THREE.CylinderGeometry(0.3, 0.3, 0.6, 8);
                break;
            case 'validation':
                geometry = new THREE.OctahedronGeometry(0.4, 0);
                break;
            default:
                geometry = new THREE.SphereGeometry(0.3, 16, 16);
        }
        
        const color = parseInt(componentData.color.replace('#', '0x'));
        const material = new THREE.MeshPhongMaterial({
            color: color,
            emissive: color,
            emissiveIntensity: 0.4,
            shininess: 100
        });
        
        const mesh = new THREE.Mesh(geometry, material);
        const pos = componentData.position;
        mesh.position.set(pos.x, level * 3 + 0.5, pos.z);
        
        // Add status indicator
        if (componentData.status === 'processing') {
            this.addProcessingIndicator(mesh);
        }
        
        group.add(mesh);
        
        // Add label
        this.createComponentLabel(componentData.label, mesh.position);
        
        return group;
    }
    
    /**
     * Add processing animation indicator
     */
    addProcessingIndicator(mesh) {
        const ringGeometry = new THREE.TorusGeometry(0.6, 0.05, 8, 32);
        const ringMaterial = new THREE.MeshBasicMaterial({
            color: 0xf59e0b,
            transparent: true,
            opacity: 0.6
        });
        
        const ring = new THREE.Mesh(ringGeometry, ringMaterial);
        ring.rotation.x = Math.PI / 2;
        ring.userData.rotating = true;
        mesh.add(ring);
    }
    
    /**
     * Create connection between components
     */
    createConnection(connectionData) {
        const sourceComp = this.components.get(connectionData.source_component);
        const targetComp = this.components.get(connectionData.target_component);
        
        if (!sourceComp || !targetComp) return;
        
        const sourceMesh = sourceComp.children[0];
        const targetMesh = targetComp.children[0];
        
        // Create curved connection
        const start = sourceMesh.position.clone();
        const end = targetMesh.position.clone();
        const mid = new THREE.Vector3(
            (start.x + end.x) / 2,
            (start.y + end.y) / 2 + 0.5,
            (start.z + end.z) / 2
        );
        
        const curve = new THREE.QuadraticBezierCurve3(start, mid, end);
        const points = curve.getPoints(50);
        const geometry = new THREE.BufferGeometry().setFromPoints(points);
        
        // Material based on connection type
        const color = this.getConnectionColor(connectionData.type);
        const material = new THREE.LineBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.7,
            linewidth: 2
        });
        
        const line = new THREE.Line(geometry, material);
        line.userData = connectionData;
        
        if (connectionData.animated) {
            line.userData.animated = true;
            line.userData.animationOffset = Math.random() * Math.PI * 2;
        }
        
        this.scene.add(line);
        this.connections.push(line);
        
        // Add arrow
        const direction = new THREE.Vector3().subVectors(end, start).normalize();
        const arrowHelper = new THREE.ArrowHelper(
            direction,
            end.clone().sub(direction.multiplyScalar(0.5)),
            0.4,
            color,
            0.15,
            0.1
        );
        this.scene.add(arrowHelper);
    }
    
    /**
     * Get connection color by type
     */
    getConnectionColor(type) {
        const colors = {
            'data-flow': 0x10b981,
            'control-flow': 0x667eea,
            'dependency': 0xf59e0b
        };
        return colors[type] || 0x8b5cf6;
    }
    
    /**
     * Create component label
     */
    createComponentLabel(text, position) {
        const canvas = document.createElement('canvas');
        const context = canvas.getContext('2d');
        canvas.width = 256;
        canvas.height = 64;
        
        context.fillStyle = '#ffffff';
        context.font = 'Bold 20px Arial';
        context.textAlign = 'center';
        context.fillText(text, 128, 32);
        
        const texture = new THREE.CanvasTexture(canvas);
        const spriteMaterial = new THREE.SpriteMaterial({ 
            map: texture,
            transparent: true,
            opacity: 0.9
        });
        
        const sprite = new THREE.Sprite(spriteMaterial);
        sprite.position.copy(position);
        sprite.position.y += 0.8;
        sprite.scale.set(1.5, 0.4, 1);
        
        this.scene.add(sprite);
    }
    
    /**
     * Focus camera on visualization
     */
    focusOnVisualization() {
        if (this.layers.length === 0) return;
        
        const maxLevel = Math.max(...this.layers.map(l => l.userData.level));
        const targetY = (maxLevel * 3) / 2;
        
        this.camera.position.set(8, targetY + 5, 8);
        this.controls.target.set(0, targetY, 0);
        this.controls.update();
    }
    
    /**
     * Clear visualization
     */
    clearVisualization() {
        this.layers.forEach(layer => this.scene.remove(layer));
        this.layers = [];
        
        this.connections.forEach(conn => this.scene.remove(conn));
        this.connections = [];
        
        this.components.clear();
        
        // Remove sprites and arrows
        const toRemove = [];
        this.scene.traverse(obj => {
            if (obj.type === 'Sprite' || obj.type === 'ArrowHelper') {
                toRemove.push(obj);
            }
        });
        toRemove.forEach(obj => this.scene.remove(obj));
    }
    
    /**
     * Animation loop
     */
    animate() {
        this.animationId = requestAnimationFrame(() => this.animate());
        
        const time = Date.now() * 0.001;
        
        // Animate components
        this.components.forEach(comp => {
            const mesh = comp.children[0];
            
            // Rotate all components slowly
            mesh.rotation.y += 0.01;
            
            // Pulse processing indicators
            if (mesh.children.length > 0) {
                mesh.children.forEach(child => {
                    if (child.userData.rotating) {
                        child.rotation.z += 0.05;
                    }
                });
            }
        });
        
        // Animate connections
        this.connections.forEach(connection => {
            if (connection.userData.animated) {
                const offset = connection.userData.animationOffset || 0;
                connection.material.opacity = 0.4 + Math.sin(time * 2 + offset) * 0.3;
            }
        });
        
        this.controls.update();
        this.renderer.render(this.scene, this.camera);
    }
    
    /**
     * Handle resize
     */
    onResize() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;
        
        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(width, height);
    }
    
    /**
     * Dispose resources
     */
    dispose() {
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
        }
        
        this.renderer.dispose();
        this.controls.dispose();
        
        if (this.renderer.domElement.parentNode) {
            this.renderer.domElement.parentNode.removeChild(this.renderer.domElement);
        }
    }
}

/**
 * Generate demo graph data for different patterns
 */
export function generateDemoGraph(type) {
    switch (type) {
        case 'cascade':
            return generateCascadePattern();
        case 'delegation-chain':
            return generateDelegationChainPattern();
        case 'multisig':
            return generateMultiSigPattern();
        case 'jurisdiction':
            return generateJurisdictionPattern();
        case 'revocation':
            return generateRevocationPattern();
        case 'full-pattern':
            return generateFullPattern();
        // Protocol Flow patterns
        case 'protocol-subscription':
            return generateProtocolSubscription();
        case 'protocol-matching':
            return generateProtocolMatching();
        case 'protocol-request':
            return generateProtocolRequest();
        case 'protocol-enforcement':
            return generateProtocolEnforcement();
        case 'protocol-verification':
            return generateProtocolVerification();
        case 'protocol-full':
            return generateProtocolFull();
        case 'simple':
            return generateSimpleGraph();
        case 'multi':
            return generateMultiPartyGraph();
        default:
            return generateComplexGraph();
    }
}

/**
 * Cascade Revocation Pattern - Shows hierarchical PoA with parent-child relationships
 */
function generateCascadePattern() {
    const nodes = [
        { id: 'root-poa', label: 'Root PoA (REVOKED)', type: 'root', status: 'revoked', position: { x: 0, y: 2, z: 0 }, metadata: { cascade_trigger: true, depth: 0 } },
        { id: 'child-poa-1', label: 'Child PoA 1 (Suspended)', type: 'delegation', status: 'suspended', position: { x: -3, y: 0, z: -2 }, metadata: { parent_revoked: true, depth: 1 } },
        { id: 'child-poa-2', label: 'Child PoA 2 (Suspended)', type: 'delegation', status: 'suspended', position: { x: 3, y: 0, z: -2 }, metadata: { parent_revoked: true, depth: 1 } },
        { id: 'grandchild-1-1', label: 'Grandchild 1.1', type: 'delegation', status: 'suspended', position: { x: -4, y: -2, z: -4 }, metadata: { cascade_depth: 2, depth: 2 } },
        { id: 'grandchild-1-2', label: 'Grandchild 1.2', type: 'delegation', status: 'suspended', position: { x: -2, y: -2, z: -4 }, metadata: { cascade_depth: 2, depth: 2 } },
        { id: 'grandchild-2-1', label: 'Grandchild 2.1', type: 'delegation', status: 'suspended', position: { x: 2, y: -2, z: -4 }, metadata: { cascade_depth: 2, depth: 2 } },
        { id: 'grandchild-2-2', label: 'Grandchild 2.2', type: 'delegation', status: 'suspended', position: { x: 4, y: -2, z: -4 }, metadata: { cascade_depth: 2, depth: 2 } }
    ];
    const edges = [
        { source: 'root-poa', target: 'child-poa-1', type: 'parent-child', label: 'delegates' },
        { source: 'root-poa', target: 'child-poa-2', type: 'parent-child', label: 'delegates' },
        { source: 'child-poa-1', target: 'grandchild-1-1', type: 'parent-child', label: 'delegates' },
        { source: 'child-poa-1', target: 'grandchild-1-2', type: 'parent-child', label: 'delegates' },
        { source: 'child-poa-2', target: 'grandchild-2-1', type: 'parent-child', label: 'delegates' },
        { source: 'child-poa-2', target: 'grandchild-2-2', type: 'parent-child', label: 'delegates' }
    ];
    return { nodes, edges, stats: { total_nodes: nodes.length, total_edges: edges.length, active_nodes: 0, pending_nodes: 0, revoked_nodes: 7 } };
}

/**
 * Delegation Chain Pattern - Shows multi-level delegation chain
 */
function generateDelegationChainPattern() {
    const nodes = [
        { id: 'issuer', label: 'Issuer (Root)', type: 'root', status: 'active', position: { x: 0, y: 3, z: 0 }, metadata: { role: 'issuer', depth: 0 } },
        { id: 'delegate-1', label: 'Delegate Level 1', type: 'delegation', status: 'active', position: { x: 0, y: 2, z: -2 }, metadata: { depth: 1 } },
        { id: 'delegate-2', label: 'Delegate Level 2', type: 'delegation', status: 'active', position: { x: 0, y: 1, z: -4 }, metadata: { depth: 2 } },
        { id: 'delegate-3', label: 'Delegate Level 3', type: 'delegation', status: 'active', position: { x: 0, y: 0, z: -6 }, metadata: { depth: 3 } },
        { id: 'delegate-4', label: 'Delegate Level 4', type: 'delegation', status: 'active', position: { x: 0, y: -1, z: -8 }, metadata: { depth: 4 } },
        { id: 'end-user', label: 'End User', type: 'consumer', status: 'active', position: { x: 0, y: -2, z: -10 }, metadata: { role: 'consumer', depth: 5 } }
    ];
    const edges = [
        { source: 'issuer', target: 'delegate-1', type: 'delegation', label: 'delegates' },
        { source: 'delegate-1', target: 'delegate-2', type: 'delegation', label: 'sub-delegates' },
        { source: 'delegate-2', target: 'delegate-3', type: 'delegation', label: 'sub-delegates' },
        { source: 'delegate-3', target: 'delegate-4', type: 'delegation', label: 'sub-delegates' },
        { source: 'delegate-4', target: 'end-user', type: 'delegation', label: 'authorizes' }
    ];
    return { nodes, edges, stats: { total_nodes: nodes.length, total_edges: edges.length, active_nodes: 6, pending_nodes: 0, revoked_nodes: 0 } };
}

/**
 * Multi-Signature Pattern - Shows collective authorization
 */
function generateMultiSigPattern() {
    const nodes = [
        { id: 'multisig-poa', label: 'Multi-Sig PoA (3/5 signed)', type: 'root', status: 'pending', position: { x: 0, y: 2, z: 0 }, metadata: { threshold: '3/5', signed: 3 } },
        { id: 'signer-1', label: 'Signer 1 ✓', type: 'signer', status: 'active', position: { x: -4, y: 0, z: -2 }, metadata: { signed: true } },
        { id: 'signer-2', label: 'Signer 2 ✓', type: 'signer', status: 'active', position: { x: -2, y: 0, z: -3 }, metadata: { signed: true } },
        { id: 'signer-3', label: 'Signer 3 ✓', type: 'signer', status: 'active', position: { x: 0, y: 0, z: -4 }, metadata: { signed: true } },
        { id: 'signer-4', label: 'Signer 4', type: 'signer', status: 'pending', position: { x: 2, y: 0, z: -3 }, metadata: { signed: false } },
        { id: 'signer-5', label: 'Signer 5', type: 'signer', status: 'pending', position: { x: 4, y: 0, z: -2 }, metadata: { signed: false } },
        { id: 'resource', label: 'Protected Resource', type: 'consumer', status: 'pending', position: { x: 0, y: -2, z: 0 }, metadata: { awaiting_activation: true } }
    ];
    const edges = [
        { source: 'signer-1', target: 'multisig-poa', type: 'signature', label: 'signed' },
        { source: 'signer-2', target: 'multisig-poa', type: 'signature', label: 'signed' },
        { source: 'signer-3', target: 'multisig-poa', type: 'signature', label: 'signed' },
        { source: 'signer-4', target: 'multisig-poa', type: 'signature-pending', label: 'pending' },
        { source: 'signer-5', target: 'multisig-poa', type: 'signature-pending', label: 'pending' },
        { source: 'multisig-poa', target: 'resource', type: 'authorization', label: 'authorizes' }
    ];
    return { nodes, edges, stats: { total_nodes: nodes.length, total_edges: edges.length, active_nodes: 4, pending_nodes: 3, revoked_nodes: 0 } };
}

/**
 * Jurisdiction Pattern - Shows geographic/legal boundaries
 */
function generateJurisdictionPattern() {
    const nodes = [
        { id: 'global-issuer', label: 'Global Issuer', type: 'root', status: 'active', position: { x: 0, y: 3, z: 0 }, metadata: { jurisdiction: 'GLOBAL' } },
        { id: 'eu-delegate', label: 'EU Delegate (GDPR)', type: 'delegation', status: 'active', position: { x: -4, y: 1, z: -2 }, metadata: { jurisdiction: 'EU', gdpr: true } },
        { id: 'us-delegate', label: 'US Delegate (HIPAA)', type: 'delegation', status: 'active', position: { x: 0, y: 1, z: -2 }, metadata: { jurisdiction: 'US', hipaa: true } },
        { id: 'asia-delegate', label: 'Asia Delegate', type: 'delegation', status: 'active', position: { x: 4, y: 1, z: -2 }, metadata: { jurisdiction: 'ASIA' } },
        { id: 'eu-service-1', label: 'EU Service (Frankfurt)', type: 'consumer', status: 'active', position: { x: -5, y: -1, z: -4 }, metadata: { location: 'Frankfurt' } },
        { id: 'eu-service-2', label: 'EU Service (Paris)', type: 'consumer', status: 'active', position: { x: -3, y: -1, z: -4 }, metadata: { location: 'Paris' } },
        { id: 'us-service', label: 'US Service (Virginia)', type: 'consumer', status: 'active', position: { x: 0, y: -1, z: -4 }, metadata: { location: 'Virginia' } },
        { id: 'asia-service', label: 'Asia Service (Singapore)', type: 'consumer', status: 'active', position: { x: 4, y: -1, z: -4 }, metadata: { location: 'Singapore' } },
        { id: 'blocked-transfer', label: 'BLOCKED Cross-Border', type: 'consumer', status: 'revoked', position: { x: -4, y: -1, z: 2 }, metadata: { violation: 'jurisdiction_mismatch' } }
    ];
    const edges = [
        { source: 'global-issuer', target: 'eu-delegate', type: 'delegation', label: 'EU scope' },
        { source: 'global-issuer', target: 'us-delegate', type: 'delegation', label: 'US scope' },
        { source: 'global-issuer', target: 'asia-delegate', type: 'delegation', label: 'Asia scope' },
        { source: 'eu-delegate', target: 'eu-service-1', type: 'authorization', label: 'authorizes' },
        { source: 'eu-delegate', target: 'eu-service-2', type: 'authorization', label: 'authorizes' },
        { source: 'us-delegate', target: 'us-service', type: 'authorization', label: 'authorizes' },
        { source: 'asia-delegate', target: 'asia-service', type: 'authorization', label: 'authorizes' },
        { source: 'eu-delegate', target: 'blocked-transfer', type: 'violation', label: 'blocked' }
    ];
    return { nodes, edges, stats: { total_nodes: nodes.length, total_edges: edges.length, active_nodes: 7, pending_nodes: 0, revoked_nodes: 1 } };
}

/**
 * Revocation States Pattern - Shows different revocation states
 */
function generateRevocationPattern() {
    const nodes = [
        { id: 'revocation-root', label: 'Revocation Tree Root', type: 'root', status: 'active', position: { x: 0, y: 3, z: 0 }, metadata: { tree_size: 128 } },
        { id: 'poa-active', label: 'Active PoA', type: 'delegation', status: 'active', position: { x: -4, y: 1, z: -2 }, metadata: { state: 'active', verified: true } },
        { id: 'poa-suspended', label: 'Suspended PoA', type: 'delegation', status: 'suspended', position: { x: -2, y: 1, z: -2 }, metadata: { state: 'suspended', reason: 'under_review' } },
        { id: 'poa-revoked', label: 'Revoked PoA', type: 'delegation', status: 'revoked', position: { x: 0, y: 1, z: -2 }, metadata: { state: 'revoked', reason: 'compromised' } },
        { id: 'poa-expired', label: 'Expired PoA', type: 'delegation', status: 'revoked', position: { x: 2, y: 1, z: -2 }, metadata: { state: 'expired', expired_at: 'timestamp' } },
        { id: 'poa-pending', label: 'Pending PoA', type: 'delegation', status: 'pending', position: { x: 4, y: 1, z: -2 }, metadata: { state: 'pending', awaiting: 'signatures' } },
        { id: 'transparency-log', label: 'Transparency Log', type: 'consumer', status: 'active', position: { x: 0, y: -1, z: -4 }, metadata: { sth_timestamp: 'latest', consistency: 'verified' } }
    ];
    const edges = [
        { source: 'revocation-root', target: 'poa-active', type: 'inclusion-proof', label: 'proof' },
        { source: 'revocation-root', target: 'poa-suspended', type: 'inclusion-proof', label: 'proof' },
        { source: 'revocation-root', target: 'poa-revoked', type: 'inclusion-proof', label: 'proof' },
        { source: 'revocation-root', target: 'poa-expired', type: 'inclusion-proof', label: 'proof' },
        { source: 'revocation-root', target: 'poa-pending', type: 'inclusion-proof', label: 'proof' },
        { source: 'poa-revoked', target: 'transparency-log', type: 'revocation-entry', label: 'logged' },
        { source: 'poa-expired', target: 'transparency-log', type: 'revocation-entry', label: 'logged' }
    ];
    return { nodes, edges, stats: { total_nodes: nodes.length, total_edges: edges.length, active_nodes: 2, pending_nodes: 1, revoked_nodes: 3 } };
}

/**
 * Full Pattern - Combines all patterns
 */
function generateFullPattern() {
    const nodes = [
        { id: 'root-issuer', label: 'Root Issuer', type: 'root', status: 'active', position: { x: 0, y: 4, z: 0 }, metadata: { role: 'root' } },
        { id: 'eu-branch', label: 'EU Branch (GDPR)', type: 'delegation', status: 'active', position: { x: -6, y: 2, z: -2 }, metadata: { jurisdiction: 'EU' } },
        { id: 'us-branch', label: 'US Branch', type: 'delegation', status: 'active', position: { x: 0, y: 2, z: -2 }, metadata: { jurisdiction: 'US' } },
        { id: 'asia-branch', label: 'Asia Branch', type: 'delegation', status: 'active', position: { x: 6, y: 2, z: -2 }, metadata: { jurisdiction: 'ASIA' } },
        { id: 'multisig-eu', label: 'EU Multi-Sig (3/5)', type: 'delegation', status: 'active', position: { x: -6, y: 0, z: -4 }, metadata: { threshold: '3/5' } },
        { id: 'us-delegate-1', label: 'US Delegate L1', type: 'delegation', status: 'active', position: { x: 0, y: 0, z: -4 }, metadata: { depth: 1 } },
        { id: 'us-delegate-2', label: 'US Delegate L2', type: 'delegation', status: 'active', position: { x: 0, y: -2, z: -6 }, metadata: { depth: 2 } },
        { id: 'asia-parent', label: 'Asia Parent (REVOKED)', type: 'delegation', status: 'revoked', position: { x: 6, y: 0, z: -4 }, metadata: { cascade_trigger: true } },
        { id: 'asia-child-1', label: 'Child 1 (Suspended)', type: 'delegation', status: 'suspended', position: { x: 5, y: -2, z: -6 }, metadata: { cascade_affected: true } },
        { id: 'asia-child-2', label: 'Child 2 (Suspended)', type: 'delegation', status: 'suspended', position: { x: 7, y: -2, z: -6 }, metadata: { cascade_affected: true } },
        { id: 'eu-service', label: 'EU Service', type: 'consumer', status: 'active', position: { x: -6, y: -4, z: -8 } },
        { id: 'us-service', label: 'US Service', type: 'consumer', status: 'active', position: { x: 0, y: -4, z: -8 } },
        { id: 'transparency-log', label: 'Transparency Log', type: 'consumer', status: 'active', position: { x: 3, y: -4, z: -8 } }
    ];
    const edges = [
        { source: 'root-issuer', target: 'eu-branch', type: 'delegation', label: 'EU scope' },
        { source: 'root-issuer', target: 'us-branch', type: 'delegation', label: 'US scope' },
        { source: 'root-issuer', target: 'asia-branch', type: 'delegation', label: 'Asia scope' },
        { source: 'eu-branch', target: 'multisig-eu', type: 'delegation', label: 'multi-sig' },
        { source: 'multisig-eu', target: 'eu-service', type: 'authorization', label: 'authorizes' },
        { source: 'us-branch', target: 'us-delegate-1', type: 'delegation', label: 'delegates' },
        { source: 'us-delegate-1', target: 'us-delegate-2', type: 'delegation', label: 'sub-delegates' },
        { source: 'us-delegate-2', target: 'us-service', type: 'authorization', label: 'authorizes' },
        { source: 'asia-branch', target: 'asia-parent', type: 'delegation', label: 'delegates' },
        { source: 'asia-parent', target: 'asia-child-1', type: 'parent-child', label: 'delegates' },
        { source: 'asia-parent', target: 'asia-child-2', type: 'parent-child', label: 'delegates' },
        { source: 'asia-parent', target: 'transparency-log', type: 'revocation-entry', label: 'revoked' }
    ];
    return { nodes, edges, stats: { total_nodes: nodes.length, total_edges: edges.length, active_nodes: 8, pending_nodes: 0, revoked_nodes: 3 } };
}

// Existing simple/multi/complex functions would go here (keep existing implementations)
function generateSimpleGraph() {
    const nodes = [
        { id: 'alice', label: 'Alice', type: 'root', status: 'active', position: { x: -2, y: 0, z: 0 } },
        { id: 'bob', label: 'Bob', type: 'delegation', status: 'active', position: { x: 0, y: 0, z: -2 } },
        { id: 'charlie', label: 'Charlie', type: 'consumer', status: 'active', position: { x: 2, y: 0, z: 0 } }
    ];
    const edges = [
        { source: 'alice', target: 'bob', type: 'delegation', label: 'delegates' },
        { source: 'bob', target: 'charlie', type: 'authorization', label: 'authorizes' }
    ];
    return { nodes, edges, stats: { total_nodes: 3, total_edges: 2, active_nodes: 3, pending_nodes: 0, revoked_nodes: 0 } };
}

function generateMultiPartyGraph() {
    const nodes = [
        { id: 'org-root', label: 'Organization', type: 'root', status: 'active', position: { x: 0, y: 2, z: 0 } },
        { id: 'dept-a', label: 'Department A', type: 'delegation', status: 'active', position: { x: -3, y: 0, z: -2 } },
        { id: 'dept-b', label: 'Department B', type: 'delegation', status: 'active', position: { x: 0, y: 0, z: -2 } },
        { id: 'dept-c', label: 'Department C', type: 'delegation', status: 'active', position: { x: 3, y: 0, z: -2 } },
        { id: 'service-1', label: 'Service 1', type: 'consumer', status: 'active', position: { x: -3, y: -2, z: -4 } },
        { id: 'service-2', label: 'Service 2', type: 'consumer', status: 'active', position: { x: 0, y: -2, z: -4 } },
        { id: 'service-3', label: 'Service 3', type: 'consumer', status: 'active', position: { x: 3, y: -2, z: -4 } }
    ];
    const edges = [
        { source: 'org-root', target: 'dept-a', type: 'delegation', label: 'delegates' },
        { source: 'org-root', target: 'dept-b', type: 'delegation', label: 'delegates' },
        { source: 'org-root', target: 'dept-c', type: 'delegation', label: 'delegates' },
        { source: 'dept-a', target: 'service-1', type: 'authorization', label: 'authorizes' },
        { source: 'dept-b', target: 'service-2', type: 'authorization', label: 'authorizes' },
        { source: 'dept-c', target: 'service-3', type: 'authorization', label: 'authorizes' },
        { source: 'dept-a', target: 'service-2', type: 'authorization', label: 'shared' },
        { source: 'dept-b', target: 'service-3', type: 'authorization', label: 'shared' }
    ];
    return { nodes, edges, stats: { total_nodes: 7, total_edges: 8, active_nodes: 7, pending_nodes: 0, revoked_nodes: 0 } };
}

function generateComplexGraph() {
    const nodes = [
        { id: 'root', label: 'Root Authority', type: 'root', status: 'active', position: { x: 0, y: 3, z: 0 } },
        { id: 'branch-1', label: 'Branch 1', type: 'delegation', status: 'active', position: { x: -4, y: 1, z: -2 } },
        { id: 'branch-2', label: 'Branch 2', type: 'delegation', status: 'active', position: { x: 0, y: 1, z: -2 } },
        { id: 'branch-3', label: 'Branch 3', type: 'delegation', status: 'pending', position: { x: 4, y: 1, z: -2 } },
        { id: 'leaf-1-1', label: 'Leaf 1.1', type: 'consumer', status: 'active', position: { x: -5, y: -1, z: -4 } },
        { id: 'leaf-1-2', label: 'Leaf 1.2', type: 'consumer', status: 'active', position: { x: -3, y: -1, z: -4 } },
        { id: 'leaf-2-1', label: 'Leaf 2.1', type: 'consumer', status: 'active', position: { x: -1, y: -1, z: -4 } },
        { id: 'leaf-2-2', label: 'Leaf 2.2', type: 'consumer', status: 'revoked', position: { x: 1, y: -1, z: -4 } },
        { id: 'leaf-3-1', label: 'Leaf 3.1', type: 'consumer', status: 'pending', position: { x: 3, y: -1, z: -4 } },
        { id: 'leaf-3-2', label: 'Leaf 3.2', type: 'consumer', status: 'pending', position: { x: 5, y: -1, z: -4 } }
    ];
    const edges = [
        { source: 'root', target: 'branch-1', type: 'delegation', label: 'delegates' },
        { source: 'root', target: 'branch-2', type: 'delegation', label: 'delegates' },
        { source: 'root', target: 'branch-3', type: 'delegation', label: 'pending' },
        { source: 'branch-1', target: 'leaf-1-1', type: 'authorization', label: 'authorizes' },
        { source: 'branch-1', target: 'leaf-1-2', type: 'authorization', label: 'authorizes' },
        { source: 'branch-2', target: 'leaf-2-1', type: 'authorization', label: 'authorizes' },
        { source: 'branch-2', target: 'leaf-2-2', type: 'authorization', label: 'revoked' },
        { source: 'branch-3', target: 'leaf-3-1', type: 'authorization', label: 'pending' },
        { source: 'branch-3', target: 'leaf-3-2', type: 'authorization', label: 'pending' },
        { source: 'leaf-1-1', target: 'leaf-2-1', type: 'cross-reference', label: 'references' }
    ];
    return { nodes, edges, stats: { total_nodes: 10, total_edges: 10, active_nodes: 5, pending_nodes: 3, revoked_nodes: 1 } };
}

/**
 * ============================================================================
 * GAuth Protocol Flow Patterns - 95% RFC Compliance
 * ============================================================================
 * These patterns visualize the RFC-0111 and RFC-0115 authorization flow stages
 * Enhanced with 30 substeps (from 19) across 6 phases:
 * - Subscription: 3 substeps
 * - Matching: 7 substeps (was 4) - Added: authorization_chain, commercial_register, formal_requirements
 * - Subset/Request: 7 substeps (was 4) - Added: request_compliance, pip_query, generate_extended_token, grant_compliance
 * - Enforcement: 4 substeps
 * - Verification: 6 substeps (was 4) - Added: validate_extended_token, pvp_identity, authorization_chain_verify, formal_requirements_verify
 * - Audit: 3 substeps
 */

/**
 * Protocol Subscription Flow - Client registration and setup
 */
function generateProtocolSubscription() {
    const nodes = [
        { id: 'client', label: 'AI Agent/Client', type: 'consumer', status: 'active', position: { x: -6, y: 2, z: 0 }, metadata: { role: 'requester' } },
        { id: 'authz-server', label: 'Authorization Server', type: 'root', status: 'active', position: { x: 0, y: 2, z: 0 }, metadata: { role: 'issuer' } },
        { id: 'register', label: '1. Register Client', type: 'delegation', status: 'active', position: { x: -4, y: 0, z: -2 }, metadata: { step: 'registration', api: '/api/v1/client/register' } },
        { id: 'configure', label: '2. Configure Scopes', type: 'delegation', status: 'active', position: { x: -2, y: 0, z: -2 }, metadata: { step: 'configuration', scopes: ['read', 'write'] } },
        { id: 'credentials', label: '3. Obtain Credentials', type: 'delegation', status: 'active', position: { x: 0, y: 0, z: -2 }, metadata: { step: 'credentials', grant_type: 'client_credentials' } },
        { id: 'client-id', label: 'Client ID & Secret', type: 'consumer', status: 'active', position: { x: -6, y: -2, z: -4 }, metadata: { credential_type: 'oauth2' } }
    ];
    const edges = [
        { source: 'client', target: 'register', type: 'request', label: 'POST /register' },
        { source: 'register', target: 'authz-server', type: 'validation', label: 'validate' },
        { source: 'authz-server', target: 'configure', type: 'response', label: 'accept' },
        { source: 'configure', target: 'credentials', type: 'flow', label: 'next' },
        { source: 'credentials', target: 'client-id', type: 'issuance', label: 'issue' },
        { source: 'client-id', target: 'client', type: 'delivery', label: 'deliver' }
    ];
    return { nodes, edges, stats: { total_nodes: 6, total_edges: 6, active_nodes: 6, pending_nodes: 0, revoked_nodes: 0 } };
}

/**
 * Protocol Matching Flow - PoA validation and capability matching
 * Updated to reflect 95% RFC compliance with 7 substeps (was 4)
 */
function generateProtocolMatching() {
    const nodes = [
        { id: 'poa-def', label: 'PoA Definition (RFC-0115)', type: 'root', status: 'active', position: { x: 0, y: 4, z: 0 }, metadata: { format: 'json', version: '1.0' } },
        { id: 'validate', label: '1. Validate PoA', type: 'delegation', status: 'active', position: { x: -5, y: 2, z: -2 }, metadata: { step: 'validation', schema_check: true } },
        { id: 'capabilities', label: '2. Check AI Capabilities', type: 'delegation', status: 'active', position: { x: -3, y: 2, z: -2 }, metadata: { step: 'capabilities', ai_level: 'advanced' } },
        { id: 'jurisdiction', label: '3. Verify Jurisdiction', type: 'delegation', status: 'active', position: { x: -1, y: 2, z: -2 }, metadata: { step: 'jurisdiction', region: 'EU', gdpr: true } },
        { id: 'policies', label: '4. Match Policies', type: 'delegation', status: 'active', position: { x: 1, y: 2, z: -2 }, metadata: { step: 'policies', rbac: true } },
        { id: 'authz-chain', label: '5. Authorization Chain ✨', type: 'delegation', status: 'active', position: { x: 3, y: 2, z: -2 }, metadata: { step: 'authorization_chain', chain_valid: true } },
        { id: 'commercial-reg', label: '6. Commercial Register ✨', type: 'delegation', status: 'active', position: { x: 5, y: 2, z: -2 }, metadata: { step: 'commercial_register', verified: true } },
        { id: 'formal-req', label: '7. Formal Requirements ✨', type: 'delegation', status: 'active', position: { x: 7, y: 2, z: -2 }, metadata: { step: 'formal_requirements', compliant: true } },
        { id: 'pdp', label: 'PDP (Policy Decision Point)', type: 'root', status: 'active', position: { x: 3, y: 0, z: -4 }, metadata: { decision: 'permit', compliance: '100%' } },
        { id: 'match-result', label: 'Match Result: PERMIT ✅', type: 'consumer', status: 'active', position: { x: 3, y: -2, z: -6 }, metadata: { confidence: 0.98, rfc_compliant: true } }
    ];
    const edges = [
        { source: 'poa-def', target: 'validate', type: 'flow', label: 'submit' },
        { source: 'validate', target: 'capabilities', type: 'flow', label: 'next' },
        { source: 'capabilities', target: 'jurisdiction', type: 'flow', label: 'next' },
        { source: 'jurisdiction', target: 'policies', type: 'flow', label: 'next' },
        { source: 'policies', target: 'authz-chain', type: 'flow', label: 'next' },
        { source: 'authz-chain', target: 'commercial-reg', type: 'flow', label: 'next' },
        { source: 'commercial-reg', target: 'formal-req', type: 'flow', label: 'next' },
        { source: 'validate', target: 'pdp', type: 'input', label: 'schema OK' },
        { source: 'capabilities', target: 'pdp', type: 'input', label: 'caps OK' },
        { source: 'jurisdiction', target: 'pdp', type: 'input', label: 'region OK' },
        { source: 'policies', target: 'pdp', type: 'input', label: 'policy OK' },
        { source: 'authz-chain', target: 'pdp', type: 'input', label: 'chain OK' },
        { source: 'commercial-reg', target: 'pdp', type: 'input', label: 'reg OK' },
        { source: 'formal-req', target: 'pdp', type: 'input', label: 'formal OK' },
        { source: 'pdp', target: 'match-result', type: 'decision', label: 'decide' }
    ];
    return { nodes, edges, stats: { total_nodes: 10, total_edges: 15, active_nodes: 10, pending_nodes: 0, revoked_nodes: 0 } };
}

/**
 * Protocol Subset/Request Flow - Authorization request with scope selection
 * Updated to reflect 95% RFC compliance with 7 substeps (was 4)
 */
function generateProtocolRequest() {
    const nodes = [
        { id: 'client', label: 'AI Agent', type: 'consumer', status: 'active', position: { x: -7, y: 3, z: 0 }, metadata: { role: 'requester' } },
        { id: 'auth-request', label: '1. Create Auth Request', type: 'delegation', status: 'active', position: { x: -5, y: 2, z: -2 }, metadata: { step: 'request', response_type: 'code' } },
        { id: 'scope-selection', label: '2. Select Scope Subset', type: 'delegation', status: 'active', position: { x: -3, y: 2, z: -2 }, metadata: { step: 'scope', scopes: ['poa:read', 'poa:delegate'] } },
        { id: 'request-compliance', label: '3. Request Compliance ✨', type: 'delegation', status: 'active', position: { x: -1, y: 2, z: -2 }, metadata: { step: 'request_compliance', validated: true } },
        { id: 'pip-query', label: '4. PIP Policy Query ✨', type: 'delegation', status: 'active', position: { x: 1, y: 2, z: -2 }, metadata: { step: 'pip_query', policy_retrieved: true } },
        { id: 'pdp-decision', label: '5. PDP Decision', type: 'root', status: 'active', position: { x: 3, y: 2, z: -2 }, metadata: { step: 'decision', result: 'permit' } },
        { id: 'token-gen', label: '6. Generate Extended Token ✨', type: 'delegation', status: 'active', position: { x: 5, y: 2, z: -2 }, metadata: { step: 'generate_extended_token', type: 'jwt+claims' } },
        { id: 'grant-compliance', label: '7. Grant Compliance ✨', type: 'delegation', status: 'active', position: { x: 7, y: 2, z: -2 }, metadata: { step: 'grant_compliance', verified: true } },
        { id: 'access-token', label: 'Extended Access Token (JWT)', type: 'consumer', status: 'active', position: { x: 3, y: 0, z: -4 }, metadata: { expires_in: 3600, token_type: 'Bearer', extended_claims: true } },
        { id: 'principal', label: 'Principal (Human)', type: 'root', status: 'active', position: { x: -7, y: 1, z: -4 }, metadata: { role: 'authorizer', consent: true } }
    ];
    const edges = [
        { source: 'client', target: 'auth-request', type: 'request', label: 'POST /authorize' },
        { source: 'auth-request', target: 'scope-selection', type: 'flow', label: 'select' },
        { source: 'principal', target: 'scope-selection', type: 'consent', label: 'authorize' },
        { source: 'scope-selection', target: 'request-compliance', type: 'flow', label: 'validate' },
        { source: 'request-compliance', target: 'pip-query', type: 'flow', label: 'query' },
        { source: 'pip-query', target: 'pdp-decision', type: 'evaluation', label: 'evaluate' },
        { source: 'pdp-decision', target: 'token-gen', type: 'permit', label: 'permit' },
        { source: 'token-gen', target: 'grant-compliance', type: 'flow', label: 'verify' },
        { source: 'grant-compliance', target: 'access-token', type: 'issuance', label: 'issue' },
        { source: 'access-token', target: 'client', type: 'delivery', label: 'deliver' }
    ];
    return { nodes, edges, stats: { total_nodes: 10, total_edges: 10, active_nodes: 10, pending_nodes: 0, revoked_nodes: 0 } };
}

/**
 * Protocol Enforcement Flow - PEP enforcement (supply-side and demand-side)
 */
function generateProtocolEnforcement() {
    const nodes = [
        { id: 'access-token', label: 'Access Token', type: 'root', status: 'active', position: { x: 0, y: 3, z: 0 }, metadata: { type: 'jwt' } },
        { id: 'supply-pep', label: 'Supply-Side PEP', type: 'delegation', status: 'active', position: { x: -4, y: 1, z: -2 }, metadata: { side: 'supply', enforcement: 'pre-check' } },
        { id: 'demand-pep', label: 'Demand-Side PEP', type: 'delegation', status: 'active', position: { x: 4, y: 1, z: -2 }, metadata: { side: 'demand', enforcement: 'post-check' } },
        { id: 'disclosure', label: 'Disclosure Requirements', type: 'delegation', status: 'active', position: { x: -2, y: -1, z: -4 }, metadata: { required: true, level: 'full' } },
        { id: 'audit-log', label: 'Audit Logging', type: 'delegation', status: 'active', position: { x: 2, y: -1, z: -4 }, metadata: { immutable: true, merkle_tree: true } },
        { id: 'resource-access', label: 'Resource Access GRANTED', type: 'consumer', status: 'active', position: { x: 0, y: -3, z: -6 }, metadata: { resource: '/api/v1/data', method: 'GET' } },
        { id: 'ai-agent', label: 'AI Agent', type: 'consumer', status: 'active', position: { x: -6, y: 1, z: 0 }, metadata: { role: 'consumer' } },
        { id: 'resource-owner', label: 'Resource Owner', type: 'consumer', status: 'active', position: { x: 6, y: 1, z: 0 }, metadata: { role: 'owner' } }
    ];
    const edges = [
        { source: 'ai-agent', target: 'supply-pep', type: 'request', label: 'request' },
        { source: 'access-token', target: 'supply-pep', type: 'validation', label: 'validate' },
        { source: 'supply-pep', target: 'disclosure', type: 'enforcement', label: 'enforce' },
        { source: 'supply-pep', target: 'audit-log', type: 'logging', label: 'log' },
        { source: 'resource-owner', target: 'demand-pep', type: 'monitoring', label: 'monitor' },
        { source: 'access-token', target: 'demand-pep', type: 'validation', label: 'validate' },
        { source: 'demand-pep', target: 'audit-log', type: 'logging', label: 'log' },
        { source: 'disclosure', target: 'resource-access', type: 'permit', label: 'permit' },
        { source: 'demand-pep', target: 'resource-access', type: 'permit', label: 'permit' }
    ];
    return { nodes, edges, stats: { total_nodes: 8, total_edges: 9, active_nodes: 8, pending_nodes: 0, revoked_nodes: 0 } };
}

/**
 * Protocol Verification Flow - Token verification and PVP identity validation
 * Updated to reflect 95% RFC compliance with 6 substeps (was 4)
 */
function generateProtocolVerification() {
    const nodes = [
        { id: 'incoming-token', label: 'Incoming Extended Token', type: 'consumer', status: 'active', position: { x: -7, y: 3, z: 0 }, metadata: { type: 'jwt+claims' } },
        { id: 'validate-token', label: '1. Validate Extended Token ✨', type: 'delegation', status: 'active', position: { x: -5, y: 2, z: -2 }, metadata: { step: 'validate_extended_token', format_ok: true } },
        { id: 'verify-sig', label: '2. Verify Signature', type: 'delegation', status: 'active', position: { x: -3, y: 2, z: -2 }, metadata: { step: 'signature', algorithm: 'RS256' } },
        { id: 'check-revocation', label: '3. Check Revocation', type: 'delegation', status: 'active', position: { x: -1, y: 2, z: -2 }, metadata: { step: 'revocation', status: 'not_revoked' } },
        { id: 'pvp-identity', label: '4. PVP Identity Verify ✨', type: 'delegation', status: 'active', position: { x: 1, y: 2, z: -2 }, metadata: { step: 'pvp_identity', identity_verified: true } },
        { id: 'authz-chain-verify', label: '5. Authorization Chain Verify ✨', type: 'delegation', status: 'active', position: { x: 3, y: 2, z: -2 }, metadata: { step: 'authorization_chain_verify', chain_valid: true } },
        { id: 'formal-req-verify', label: '6. Formal Requirements Verify ✨', type: 'delegation', status: 'active', position: { x: 5, y: 2, z: -2 }, metadata: { step: 'formal_requirements_verify', compliant: true } },
        { id: 'jwks', label: 'JWKS Endpoint', type: 'root', status: 'active', position: { x: -3, y: 4, z: -4 }, metadata: { keys: 'public_keys' } },
        { id: 'revocation-list', label: 'Revocation List', type: 'root', status: 'active', position: { x: -1, y: 4, z: -4 }, metadata: { type: 'merkle_tree' } },
        { id: 'pvp-registry', label: 'PVP Registry', type: 'root', status: 'active', position: { x: 1, y: 4, z: -4 }, metadata: { verification: 'did_method' } },
        { id: 'verification-result', label: 'Verification: VALID ✅', type: 'consumer', status: 'active', position: { x: 1, y: 0, z: -6 }, metadata: { result: 'valid', confidence: 1.0, rfc_compliant: true } }
    ];
    const edges = [
        { source: 'incoming-token', target: 'validate-token', type: 'flow', label: 'submit' },
        { source: 'validate-token', target: 'verify-sig', type: 'flow', label: 'next' },
        { source: 'verify-sig', target: 'jwks', type: 'lookup', label: 'fetch_key' },
        { source: 'verify-sig', target: 'check-revocation', type: 'flow', label: 'next' },
        { source: 'check-revocation', target: 'revocation-list', type: 'lookup', label: 'check' },
        { source: 'check-revocation', target: 'pvp-identity', type: 'flow', label: 'next' },
        { source: 'pvp-identity', target: 'pvp-registry', type: 'lookup', label: 'verify' },
        { source: 'pvp-identity', target: 'authz-chain-verify', type: 'flow', label: 'next' },
        { source: 'authz-chain-verify', target: 'formal-req-verify', type: 'flow', label: 'next' },
        { source: 'formal-req-verify', target: 'verification-result', type: 'decision', label: 'valid' }
    ];
    return { nodes, edges, stats: { total_nodes: 11, total_edges: 10, active_nodes: 11, pending_nodes: 0, revoked_nodes: 0 } };
}

/**
 * Complete Protocol Flow - Full RFC-0111 + RFC-0115 end-to-end flow
 * 95% RFC Compliant - 30 substeps total across 6 phases
 */
function generateProtocolFull() {
    const nodes = [
        // Subscription Phase (3 substeps)
        { id: 'client', label: 'AI Agent', type: 'consumer', status: 'active', position: { x: -8, y: 4, z: 0 }, metadata: { phase: 'subscription', substeps: 3 } },
        { id: 'subscription', label: 'Subscription (3 steps)', type: 'delegation', status: 'active', position: { x: -6, y: 4, z: -2 }, metadata: { phase: 'subscription', icon: '📝' } },
        
        // Matching Phase (7 substeps - was 4) ✨
        { id: 'poa-def', label: 'PoA Definition', type: 'root', status: 'active', position: { x: -4, y: 4, z: -4 }, metadata: { phase: 'matching', format: 'rfc-0115' } },
        { id: 'matching', label: 'Matching (7 steps) ✨', type: 'delegation', status: 'active', position: { x: -2, y: 4, z: -6 }, metadata: { phase: 'matching', icon: '🔍', substeps: 7, enhanced: true } },
        
        // Request Phase (7 substeps - was 4) ✨
        { id: 'principal', label: 'Principal (Human)', type: 'root', status: 'active', position: { x: 0, y: 6, z: -8 }, metadata: { phase: 'request', role: 'authorizer' } },
        { id: 'request', label: 'Subset/Request (7 steps) ✨', type: 'delegation', status: 'active', position: { x: 0, y: 4, z: -8 }, metadata: { phase: 'request', icon: '🎯', substeps: 7, enhanced: true } },
        { id: 'access-token', label: 'Extended Access Token ✨', type: 'consumer', status: 'active', position: { x: 2, y: 4, z: -10 }, metadata: { phase: 'request', type: 'jwt+claims' } },
        
        // Enforcement Phase (4 substeps)
        { id: 'supply-pep', label: 'Supply PEP', type: 'delegation', status: 'active', position: { x: 4, y: 2, z: -12 }, metadata: { phase: 'enforcement', side: 'supply', substeps: 4 } },
        { id: 'demand-pep', label: 'Demand PEP', type: 'delegation', status: 'active', position: { x: 6, y: 2, z: -12 }, metadata: { phase: 'enforcement', side: 'demand' } },
        
        // Verification Phase (6 substeps - was 4) ✨
        { id: 'verification', label: 'Verification (6 steps) ✨', type: 'delegation', status: 'active', position: { x: 4, y: 0, z: -14 }, metadata: { phase: 'verification', icon: '✓', substeps: 6, enhanced: true } },
        
        // Audit Phase (3 substeps)
        { id: 'audit', label: 'Audit & Compliance (3 steps)', type: 'delegation', status: 'active', position: { x: 2, y: -2, z: -16 }, metadata: { phase: 'audit', icon: '📊', substeps: 3 } },
        { id: 'resource', label: 'Protected Resource ✅', type: 'consumer', status: 'active', position: { x: 0, y: -2, z: -18 }, metadata: { access: 'granted', rfc_compliant: true } },
        
        // Supporting Services
        { id: 'authz-server', label: 'Authorization Server', type: 'root', status: 'active', position: { x: -8, y: 0, z: -8 }, metadata: { role: 'issuer' } },
        { id: 'pdp', label: 'PDP (Enhanced)', type: 'root', status: 'active', position: { x: -6, y: 0, z: -10 }, metadata: { decision: 'permit', compliance: '95%' } },
        { id: 'transparency-log', label: 'Transparency Log', type: 'root', status: 'active', position: { x: 0, y: -4, z: -16 }, metadata: { immutable: true } },
        
        // New Enhanced Components ✨
        { id: 'authz-chain-validator', label: 'Authorization Chain ✨', type: 'root', status: 'active', position: { x: -4, y: 2, z: -8 }, metadata: { feature: 'authz_chain' } },
        { id: 'commercial-reg', label: 'Commercial Register ✨', type: 'root', status: 'active', position: { x: -2, y: 2, z: -8 }, metadata: { feature: 'commercial_register' } },
        { id: 'pip-service', label: 'PIP (Policy Info) ✨', type: 'root', status: 'active', position: { x: 2, y: 2, z: -10 }, metadata: { feature: 'pip_query' } }
    ];
    
    const edges = [
        // Subscription flow
        { source: 'client', target: 'subscription', type: 'flow', label: 'register' },
        { source: 'subscription', target: 'authz-server', type: 'registration', label: 'register' },
        
        // Matching flow (enhanced with 7 substeps)
        { source: 'poa-def', target: 'matching', type: 'flow', label: 'validate' },
        { source: 'matching', target: 'pdp', type: 'validation', label: 'match' },
        { source: 'matching', target: 'authz-chain-validator', type: 'validation', label: 'validate chain' },
        { source: 'matching', target: 'commercial-reg', type: 'validation', label: 'check register' },
        
        // Request flow (enhanced with 7 substeps)
        { source: 'principal', target: 'request', type: 'consent', label: 'authorize' },
        { source: 'request', target: 'pip-service', type: 'query', label: 'query policy' },
        { source: 'request', target: 'pdp', type: 'evaluation', label: 'evaluate' },
        { source: 'request', target: 'access-token', type: 'issuance', label: 'issue extended' },
        
        // Enforcement flow
        { source: 'access-token', target: 'supply-pep', type: 'validation', label: 'validate' },
        { source: 'access-token', target: 'demand-pep', type: 'validation', label: 'validate' },
        { source: 'supply-pep', target: 'verification', type: 'flow', label: 'verify' },
        { source: 'demand-pep', target: 'verification', type: 'flow', label: 'verify' },
        
        // Verification flow (enhanced with 6 substeps)
        { source: 'verification', target: 'authz-chain-validator', type: 'validation', label: 're-verify chain' },
        { source: 'verification', target: 'audit', type: 'flow', label: 'log' },
        
        // Audit flow
        { source: 'audit', target: 'transparency-log', type: 'logging', label: 'record' },
        { source: 'audit', target: 'resource', type: 'permit', label: 'grant_access' },
        
        // Cross-phase connections
        { source: 'subscription', target: 'poa-def', type: 'flow', label: 'next' },
        { source: 'matching', target: 'request', type: 'flow', label: 'next' }
    ];
    
    return { nodes, edges, stats: { total_nodes: 15, total_edges: 17, active_nodes: 15, pending_nodes: 0, revoked_nodes: 0 } };
}
