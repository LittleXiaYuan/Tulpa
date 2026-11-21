const api = process.env.TULPA_API_URL || 'http://127.0.0.1:8080';

export default {
  devIndicators: false,
  async rewrites() {
    return [{ source: '/api/:path*', destination: api + '/api/:path*' }];
  },
};
