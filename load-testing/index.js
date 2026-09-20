import http from 'k6/http';

export const options = {
  discardResponseBodies: true,
  duration: '60s',
  vus: 1000,
  // executor: 'shared-iterations',
};

export default function () {
  http.get('http://127.0.0.1:3000/request');
}