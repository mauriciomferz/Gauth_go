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
 * GAuth Protocol Flow Patterns
 * ============================================================================
 * These patterns visualize the RFC-0111 and RFC-0115 authorization flow stages
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
 */
function generateProtocolMatching() {
    const nodes = [
        { id: 'poa-def', label: 'PoA Definition (RFC-0115)', type: 'root', status: 'active', position: { x: 0, y: 3, z: 0 }, metadata: { format: 'json', version: '1.0' } },
        { id: 'validate', label: '1. Validate PoA', type: 'delegation', status: 'active', position: { x: -4, y: 1, z: -2 }, metadata: { step: 'validation', schema_check: true } },
        { id: 'capabilities', label: '2. Check AI Capabilities', type: 'delegation', status: 'active', position: { x: -2, y: 1, z: -2 }, metadata: { step: 'capabilities', ai_level: 'advanced' } },
        { id: 'jurisdiction', label: '3. Verify Jurisdiction', type: 'delegation', status: 'active', position: { x: 0, y: 1, z: -2 }, metadata: { step: 'jurisdiction', region: 'EU', gdpr: true } },
        { id: 'policies', label: '4. Match Policies', type: 'delegation', status: 'active', position: { x: 2, y: 1, z: -2 }, metadata: { step: 'policies', rbac: true } },
        { id: 'pdp', label: 'PDP (Policy Decision Point)', type: 'root', status: 'active', position: { x: 0, y: -1, z: -4 }, metadata: { decision: 'permit' } },
        { id: 'match-result', label: 'Match Result: PERMIT', type: 'consumer', status: 'active', position: { x: 0, y: -3, z: -6 }, metadata: { confidence: 0.98 } }
    ];
    const edges = [
        { source: 'poa-def', target: 'validate', type: 'flow', label: 'submit' },
        { source: 'validate', target: 'capabilities', type: 'flow', label: 'next' },
        { source: 'capabilities', target: 'jurisdiction', type: 'flow', label: 'next' },
        { source: 'jurisdiction', target: 'policies', type: 'flow', label: 'next' },
        { source: 'validate', target: 'pdp', type: 'input', label: 'schema OK' },
        { source: 'capabilities', target: 'pdp', type: 'input', label: 'caps OK' },
        { source: 'jurisdiction', target: 'pdp', type: 'input', label: 'region OK' },
        { source: 'policies', target: 'pdp', type: 'input', label: 'policy OK' },
        { source: 'pdp', target: 'match-result', type: 'decision', label: 'decide' }
    ];
    return { nodes, edges, stats: { total_nodes: 7, total_edges: 9, active_nodes: 7, pending_nodes: 0, revoked_nodes: 0 } };
}

/**
 * Protocol Subset/Request Flow - Authorization request with scope selection
 */
function generateProtocolRequest() {
    const nodes = [
        { id: 'client', label: 'AI Agent', type: 'consumer', status: 'active', position: { x: -6, y: 2, z: 0 }, metadata: { role: 'requester' } },
        { id: 'auth-request', label: '1. Create Auth Request', type: 'delegation', status: 'active', position: { x: -3, y: 1, z: -2 }, metadata: { step: 'request', response_type: 'code' } },
        { id: 'scope-selection', label: '2. Select Scope Subset', type: 'delegation', status: 'active', position: { x: 0, y: 1, z: -2 }, metadata: { step: 'scope', scopes: ['poa:read', 'poa:delegate'] } },
        { id: 'pdp-decision', label: '3. PDP Decision', type: 'root', status: 'active', position: { x: 3, y: 1, z: -2 }, metadata: { step: 'decision', result: 'permit' } },
        { id: 'token-gen', label: '4. Generate Token', type: 'delegation', status: 'active', position: { x: 3, y: -1, z: -4 }, metadata: { step: 'token', type: 'jwt' } },
        { id: 'access-token', label: 'Access Token (JWT)', type: 'consumer', status: 'active', position: { x: 0, y: -2, z: -6 }, metadata: { expires_in: 3600, token_type: 'Bearer' } },
        { id: 'principal', label: 'Principal (Human)', type: 'root', status: 'active', position: { x: -6, y: 0, z: -4 }, metadata: { role: 'authorizer', consent: true } }
    ];
    const edges = [
        { source: 'client', target: 'auth-request', type: 'request', label: 'POST /authorize' },
        { source: 'auth-request', target: 'scope-selection', type: 'flow', label: 'select' },
        { source: 'principal', target: 'scope-selection', type: 'consent', label: 'authorize' },
        { source: 'scope-selection', target: 'pdp-decision', type: 'evaluation', label: 'evaluate' },
        { source: 'pdp-decision', target: 'token-gen', type: 'permit', label: 'permit' },
        { source: 'token-gen', target: 'access-token', type: 'issuance', label: 'issue' },
        { source: 'access-token', target: 'client', type: 'delivery', label: 'deliver' }
    ];
    return { nodes, edges, stats: { total_nodes: 7, total_edges: 7, active_nodes: 7, pending_nodes: 0, revoked_nodes: 0 } };
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
 */
function generateProtocolVerification() {
    const nodes = [
        { id: 'incoming-token', label: 'Incoming Token', type: 'consumer', status: 'active', position: { x: -6, y: 2, z: 0 }, metadata: { type: 'jwt' } },
        { id: 'validate-token', label: '1. Validate Token', type: 'delegation', status: 'active', position: { x: -3, y: 0, z: -2 }, metadata: { step: 'validation', format_ok: true } },
        { id: 'verify-sig', label: '2. Verify Signature', type: 'delegation', status: 'active', position: { x: -1, y: 0, z: -2 }, metadata: { step: 'signature', algorithm: 'RS256' } },
        { id: 'check-revocation', label: '3. Check Revocation', type: 'delegation', status: 'active', position: { x: 1, y: 0, z: -2 }, metadata: { step: 'revocation', status: 'not_revoked' } },
        { id: 'pvp-check', label: '4. PVP Identity Check', type: 'delegation', status: 'active', position: { x: 3, y: 0, z: -2 }, metadata: { step: 'pvp', identity_verified: true } },
        { id: 'jwks', label: 'JWKS Endpoint', type: 'root', status: 'active', position: { x: -1, y: 2, z: -4 }, metadata: { keys: 'public_keys' } },
        { id: 'revocation-list', label: 'Revocation List', type: 'root', status: 'active', position: { x: 1, y: 2, z: -4 }, metadata: { type: 'merkle_tree' } },
        { id: 'pvp-registry', label: 'PVP Registry', type: 'root', status: 'active', position: { x: 3, y: 2, z: -4 }, metadata: { verification: 'did_method' } },
        { id: 'verification-result', label: 'Verification: VALID', type: 'consumer', status: 'active', position: { x: 0, y: -2, z: -6 }, metadata: { result: 'valid', confidence: 1.0 } }
    ];
    const edges = [
        { source: 'incoming-token', target: 'validate-token', type: 'flow', label: 'submit' },
        { source: 'validate-token', target: 'verify-sig', type: 'flow', label: 'next' },
        { source: 'verify-sig', target: 'jwks', type: 'lookup', label: 'fetch_key' },
        { source: 'verify-sig', target: 'check-revocation', type: 'flow', label: 'next' },
        { source: 'check-revocation', target: 'revocation-list', type: 'lookup', label: 'check' },
        { source: 'check-revocation', target: 'pvp-check', type: 'flow', label: 'next' },
        { source: 'pvp-check', target: 'pvp-registry', type: 'lookup', label: 'verify' },
        { source: 'pvp-check', target: 'verification-result', type: 'decision', label: 'valid' }
    ];
    return { nodes, edges, stats: { total_nodes: 9, total_edges: 8, active_nodes: 9, pending_nodes: 0, revoked_nodes: 0 } };
}

/**
 * Complete Protocol Flow - Full RFC-0111 + RFC-0115 end-to-end flow
 */
function generateProtocolFull() {
    const nodes = [
        // Subscription Phase
        { id: 'client', label: 'AI Agent', type: 'consumer', status: 'active', position: { x: -8, y: 4, z: 0 }, metadata: { phase: 'subscription' } },
        { id: 'subscription', label: 'Subscription', type: 'delegation', status: 'active', position: { x: -6, y: 4, z: -2 }, metadata: { phase: 'subscription', icon: '📝' } },
        
        // Matching Phase
        { id: 'poa-def', label: 'PoA Definition', type: 'root', status: 'active', position: { x: -4, y: 4, z: -4 }, metadata: { phase: 'matching', format: 'rfc-0115' } },
        { id: 'matching', label: 'Matching', type: 'delegation', status: 'active', position: { x: -2, y: 4, z: -6 }, metadata: { phase: 'matching', icon: '🔍' } },
        
        // Request Phase
        { id: 'principal', label: 'Principal (Human)', type: 'root', status: 'active', position: { x: 0, y: 6, z: -8 }, metadata: { phase: 'request', role: 'authorizer' } },
        { id: 'request', label: 'Subset/Request', type: 'delegation', status: 'active', position: { x: 0, y: 4, z: -8 }, metadata: { phase: 'request', icon: '🎯' } },
        { id: 'access-token', label: 'Access Token', type: 'consumer', status: 'active', position: { x: 2, y: 4, z: -10 }, metadata: { phase: 'request', type: 'jwt' } },
        
        // Enforcement Phase
        { id: 'supply-pep', label: 'Supply PEP', type: 'delegation', status: 'active', position: { x: 4, y: 2, z: -12 }, metadata: { phase: 'enforcement', side: 'supply' } },
        { id: 'demand-pep', label: 'Demand PEP', type: 'delegation', status: 'active', position: { x: 6, y: 2, z: -12 }, metadata: { phase: 'enforcement', side: 'demand' } },
        
        // Verification Phase
        { id: 'verification', label: 'Verification', type: 'delegation', status: 'active', position: { x: 4, y: 0, z: -14 }, metadata: { phase: 'verification', icon: '✓' } },
        
        // Audit Phase
        { id: 'audit', label: 'Audit & Compliance', type: 'delegation', status: 'active', position: { x: 2, y: -2, z: -16 }, metadata: { phase: 'audit', icon: '📊' } },
        { id: 'resource', label: 'Protected Resource', type: 'consumer', status: 'active', position: { x: 0, y: -2, z: -18 }, metadata: { access: 'granted' } },
        
        // Supporting Services
        { id: 'authz-server', label: 'Authorization Server', type: 'root', status: 'active', position: { x: -8, y: 0, z: -8 }, metadata: { role: 'issuer' } },
        { id: 'pdp', label: 'PDP', type: 'root', status: 'active', position: { x: -6, y: 0, z: -10 }, metadata: { decision: 'permit' } },
        { id: 'transparency-log', label: 'Transparency Log', type: 'root', status: 'active', position: { x: 0, y: -4, z: -16 }, metadata: { immutable: true } }
    ];
    
    const edges = [
        // Subscription flow
        { source: 'client', target: 'subscription', type: 'flow', label: 'register' },
        { source: 'subscription', target: 'authz-server', type: 'registration', label: 'register' },
        
        // Matching flow
        { source: 'poa-def', target: 'matching', type: 'flow', label: 'validate' },
        { source: 'matching', target: 'pdp', type: 'validation', label: 'match' },
        
        // Request flow
        { source: 'principal', target: 'request', type: 'consent', label: 'authorize' },
        { source: 'request', target: 'pdp', type: 'evaluation', label: 'evaluate' },
        { source: 'request', target: 'access-token', type: 'issuance', label: 'issue' },
        
        // Enforcement flow
        { source: 'access-token', target: 'supply-pep', type: 'validation', label: 'validate' },
        { source: 'access-token', target: 'demand-pep', type: 'validation', label: 'validate' },
        { source: 'supply-pep', target: 'verification', type: 'flow', label: 'verify' },
        { source: 'demand-pep', target: 'verification', type: 'flow', label: 'verify' },
        
        // Verification flow
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
