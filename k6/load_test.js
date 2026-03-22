import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    scenarios: {
        constant_request_rate: {
            executor: 'constant-arrival-rate',
            rate: __ENV.RPS || 100, // Passado via variável de ambiente, default 100
            timeUnit: '1s',         // RPS (Requests Per Second)
            duration: '1m',         // Duração do teste
            preAllocatedVUs: 1000,  // Pre-aloca um bom número de VUs para suportar a carga
            maxVUs: 15000,          // Limite máximo de VUs caso as requests comecem a demorar
        },
    },
};

export default function() {
    let url = 'http://host.docker.internal:8080/ingest';
    
    let payload = JSON.stringify({
        device_id: `device_${Math.floor(Math.random() * 1000)}`,
        timestamp: new Date().toISOString(),
        sensor_type: Math.random() > 0.5 ? 'temperature' : 'presence',
        reading_nature: Math.random() > 0.5 ? 'analog' : 'discrete',
        value: Math.random() * 100,
    });
    
    let params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    let res = http.post(url, payload, params);

    check(res, {
        'status is 202': (r) => r.status === 202,
    });
}