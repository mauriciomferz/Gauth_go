/**
 * PoA Visualization Module
 * Think Machine-inspired 3D visualization for Power of Attorney relationships
 * and protocol step flows
 */

import * as THREE from 'https://cdn.jsdelivr.net/npm/three@0.160.0/build/three.module.js';
import { OrbitControls } from 'https://cdn.jsdelivr.net/npm/three@0.160.0/examples/jsm/controls/OrbitControls.js';

/**
 * PoA Graph Visualizer
 * Renders PoA relationship graphs in 3D space
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
        this.animationId = null;
        
        this.init();
    }
    
    init() {
        // Scene
        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x0a0a0a);
        this.scene.fog = new THREE.Fog(0x0a0a0a, 10, 50);
        
        // Camera
        const aspect = this.container.clientWidth / this.container.clientHeight;
        this.camera = new THREE.PerspectiveCamera(60, aspect, 0.1, 1000);
        this.camera.position.set(10, 10, 10);
        
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
        this.controls.screenSpacePanning = false;
        this.controls.minDistance = 3;
        this.controls.maxDistance = 50;
        
        // Lighting
        this.setupLighting();
        
        // Grid
        const gridHelper = new THREE.GridHelper(20, 20, 0x444444, 0x222222);
        this.scene.add(gridHelper);
        
        // Handle resize
        window.addEventListener('resize', () => this.onResize());
        
        // Start animation loop
        this.animate();
    }
    
    setupLighting() {
        // Ambient light
        const ambientLight = new THREE.AmbientLight(0x404040, 0.5);
        this.scene.add(ambientLight);
        
        // Point lights for dramatic effect (Think Machine style)
        const light1 = new THREE.PointLight(0x667eea, 1, 50);
        light1.position.set(10, 10, 10);
        this.scene.add(light1);
        
        const light2 = new THREE.PointLight(0xf59e0b, 0.8, 50);
        light2.position.set(-10, 10, -10);
        this.scene.add(light2);
        
        const light3 = new THREE.PointLight(0x10b981, 0.6, 50);
        light3.position.set(0, -10, 0);
        this.scene.add(light3);
    }
    
    /**
     * Load and render a PoA graph
     */
    async loadGraph(graphData) {
        // Clear existing visualization
        this.clearGraph();
        
        // Create nodes
        for (const node of graphData.nodes) {
            this.createNode(node);
        }
        
        // Create edges
        for (const edge of graphData.edges) {
            this.createEdge(edge);
        }
    }
    
    /**
     * Create a 3D node representation
     */
    createNode(nodeData) {
        const position = nodeData.position || { x: 0, y: 0, z: 0 };
        
        // Node geometry based on type
        let geometry;
        switch (nodeData.type) {
            case 'principal':
                geometry = new THREE.OctahedronGeometry(0.5, 0);
                break;
            case 'client':
                geometry = new THREE.TetrahedronGeometry(0.5, 0);
                break;
            case 'resource':
                geometry = new THREE.BoxGeometry(0.6, 0.6, 0.6);
                break;
            default:
                geometry = new THREE.SphereGeometry(0.4, 16, 16);
        }
        
        // Material with status-based color
        const color = this.getStatusColor(nodeData.status);
        const material = new THREE.MeshPhongMaterial({
            color: color,
            emissive: color,
            emissiveIntensity: 0.3,
            shininess: 100,
            transparent: true,
            opacity: 0.9
        });
        
        const mesh = new THREE.Mesh(geometry, material);
        mesh.position.set(position.x, position.y, position.z);
        mesh.userData = nodeData;
        
        // Add glow effect
        const glowGeometry = geometry.clone();
        const glowMaterial = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.2,
            side: THREE.BackSide
        });
        const glow = new THREE.Mesh(glowGeometry, glowMaterial);
        glow.scale.multiplyScalar(1.3);
        mesh.add(glow);
        
        // Add to scene
        this.scene.add(mesh);
        this.nodes.set(nodeData.id, mesh);
        
        // Add label
        this.createLabel(nodeData.label, position);
        
        return mesh;
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
     * Clear the graph
     */
    clearGraph() {
        // Remove nodes
        this.nodes.forEach(node => {
            this.scene.remove(node);
        });
        this.nodes.clear();
        
        // Remove edges
        this.edges.forEach(edge => {
            this.scene.remove(edge);
        });
        this.edges = [];
        
        // Clear other objects except grid and lights
        const objectsToRemove = [];
        this.scene.traverse(obj => {
            if (obj.type === 'Sprite' || obj.type === 'ArrowHelper') {
                objectsToRemove.push(obj);
            }
        });
        objectsToRemove.forEach(obj => this.scene.remove(obj));
    }
    
    /**
     * Animation loop
     */
    animate() {
        this.animationId = requestAnimationFrame(() => this.animate());
        
        // Rotate nodes slightly for visual effect
        this.nodes.forEach(node => {
            node.rotation.y += 0.002;
            
            // Pulse effect for pending nodes
            if (node.userData.status === 'pending') {
                const scale = 1 + Math.sin(Date.now() * 0.003) * 0.1;
                node.scale.set(scale, scale, scale);
            }
        });
        
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
        this.scene.background = new THREE.Color(0x0a0a0a);
        
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
        const ambientLight = new THREE.AmbientLight(0x404040, 0.6);
        this.scene.add(ambientLight);
        
        const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
        directionalLight.position.set(5, 10, 5);
        this.scene.add(directionalLight);
        
        // Accent lights
        const accentLight1 = new THREE.PointLight(0x667eea, 0.5, 20);
        accentLight1.position.set(5, 5, 5);
        this.scene.add(accentLight1);
        
        const accentLight2 = new THREE.PointLight(0xf59e0b, 0.5, 20);
        accentLight2.position.set(-5, 5, -5);
        this.scene.add(accentLight2);
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
