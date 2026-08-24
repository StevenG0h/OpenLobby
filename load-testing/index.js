import http from 'k6/http';

export const options = {
  discardResponseBodies: true,
      duration: '600s',
      vus: 150,
};

export default function () {
  http.get('http://192.168.1.20:3000');
}