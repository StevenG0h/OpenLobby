import http from 'k6/http';

export const options = {
  discardResponseBodies: true,
      duration: '60s',
      vus: 1000,
};

export default function () {
  http.get('http://localhost:3000/request');
}