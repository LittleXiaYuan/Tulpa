import './style.css';

export const metadata = {
  title: 'Tulpa Terminal',
  description: 'OKX spot market dashboard and quantitative strategy research.',
};

export default function Layout({ children }) {
  return <html lang="en"><body>{children}</body></html>;
}
